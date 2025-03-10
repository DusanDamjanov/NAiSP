package SSTable

import (
	. "NAiSP/BloomFilter"
	. "NAiSP/Log"
	. "NAiSP/merkleTree"
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

//const (
//	SUMMARY_BLOCK_SIZE = 10
//)

type SSTable struct {
	Header     Header
	Generation int
	Data       []Log
	Index      Index
	Summary    Summary
	Filter     Bloom
	//	TOC        TOCEntry
	Metadata MerkleRoot
}

func PrintLogs(fileType string, generation string, level string) {
	fmt.Println("-------------------------------")
	fmt.Println("SSTable -", generation, "-", level)
	fmt.Println("-------------------------------")
	file, err := os.Open("./Data/SSTables/" + fileType + "/Data-" + generation + "-" + level + ".bin")
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	logs, _ := GetAllLogs(file, fileType)
	for _, log := range logs {
		fmt.Println("Key-", string(log.Key), "  ", "Value-", string(log.Value), log.Tombstone, "Time-", log.Timestamp)
	}
}

func GetALLLevels(dirPath string) []int {
	var levels []int

	// Read the directory and get a list of file and folder names
	fileInfos, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return nil
	}

	//find files from same level of LSM tree
	for _, fileInfo := range fileInfos {
		numbers := strings.Split(fileInfo.Name(), "-")
		fileLevelSplit := strings.Split(numbers[2], ".")
		fileLevel, err := strconv.Atoi(fileLevelSplit[0])
		generation, err := strconv.Atoi(numbers[1])
		if err != nil {
			fmt.Println("Error, wrong file format:", err)
			return nil
		}
		if generation == 2 {
			levels = append(levels, fileLevel)
		}
	}

	return levels
}
func ContainsElement(slice []int, element int) bool {
	for _, value := range slice {
		if value == element {
			return true
		}
	}
	return false
}

func BuildSSTable(sortedData []*Log, generation int, level int, sstableType string, summaryBlockSize int) {
	if sstableType == "single" {
		BuildSSTableSingle(sortedData, generation, level, summaryBlockSize)
		return
	}
	BuildSSTableMultiple(sortedData, generation, level, summaryBlockSize)
}

func GetAllLogs(file *os.File, sstableType string) ([]*Log, error) {
	var data []*Log
	if sstableType == "single" {
		header, _ := ReadHeader(file)
		data, err := ReadLogs(file, int64(header.LogsOffset), header.BloomOffset)
		if err != nil {
			fmt.Println("Error reading logs from single file")
			return nil, err
		}
		return data, nil
	}
	//for Multiple
	offsetEnd, _ := file.Seek(0, os.SEEK_END)
	data, err := ReadLogs(file, 0, uint64(offsetEnd))
	if err != nil {
		fmt.Println("Error reading logs from multiple file")
		return nil, err
	}

	return data, nil
}

// MULTIPLE:
func BuildSSTableMultiple(sortedData []*Log, generation, level, SUMMARY_BLOCK_SIZE int) {
	//cetri bafera za cetri razlicita fajla
	var FilterContent = new(bytes.Buffer)
	var DataContent = new(bytes.Buffer)
	var IndexContent = new(bytes.Buffer)
	var SummaryContent = new(bytes.Buffer)
	TOCData := ""

	filter := BuildFilter(sortedData, len(sortedData), 0.01)
	binary.Write(FilterContent, binary.LittleEndian, filter.Serialize().Bytes())

	var offsetLog uint64
	offsetLog = 0
	WriteSummaryHeader(sortedData, SummaryContent) //u summary ce ispisati prvi i poslednji kljuc iz indexa
	for i, data := range sortedData {              //za svaki podatak
		binary.Write(DataContent, binary.LittleEndian, data.Serialize()) //ubaci ga u baffer
		if ((i+1)%SUMMARY_BLOCK_SIZE) == 0 || i == 0 {                   //svaki 10. kljuc - summary napravljen u fazonu da ima jos indexa ne samo prvi i poslednji
			WriteSummaryLog(SummaryContent, uint64(sortedData[i].KeySize), sortedData[i].Key, uint64(IndexContent.Len()))
			//kako indexEntry i dalje nije zapisan pocetak njega je trenutna duzina indexcontent buffera, dakle ubacujemo ga u summary
		}
		WriteIndexLog(IndexContent, uint64(data.KeySize), data.Key, offsetLog) //tek sad pisemo indexEntry u index bafer
		offsetLog += uint64(len(data.Serialize()))
	}

	merkle := BuildMerkleTreeRoot(sortedData)
	//fje koje ce kreirati fajlove i ispisati sadrzaj navedenih bafera
	WriteToFile(generation, level, "Data", "Multiple", DataContent, &TOCData)
	WriteToFile(generation, level, "Index", "Multiple", IndexContent, &TOCData)
	WriteToFile(generation, level, "Summary", "Multiple", SummaryContent, &TOCData)
	WriteToFile(generation, level, "Bloom", "Multiple", FilterContent, &TOCData)
	WriteToTxtFile(generation, level, "Metadata", "Multiple", hex.EncodeToString(SerializeMerkleTree(merkle)), &TOCData)
	WriteToTxtFile(generation, level, "TOC", "Multiple", TOCData, nil)
}

func WriteSummaryLog(SummaryContent *bytes.Buffer, KeySize, Key, OffsetInIndexFile any) {
	binary.Write(SummaryContent, binary.LittleEndian, KeySize)           //upisi velicinu kljuca
	binary.Write(SummaryContent, binary.LittleEndian, Key)               //kljuc
	binary.Write(SummaryContent, binary.LittleEndian, OffsetInIndexFile) //trenutna duzina index bufera(kako 10. kljuc jos nije upisan ovo ce biti pocetak 10. kljuca)

}
func WriteSummaryHeader(sortedData []*Log, SummaryContent *bytes.Buffer) {
	binary.Write(SummaryContent, binary.LittleEndian, sortedData[0].KeySize) //min key
	binary.Write(SummaryContent, binary.LittleEndian, sortedData[0].Key)
	binary.Write(SummaryContent, binary.LittleEndian, sortedData[len(sortedData)-1].KeySize) //max key
	binary.Write(SummaryContent, binary.LittleEndian, sortedData[len(sortedData)-1].Key)
}

func WriteIndexLog(IndexContent *bytes.Buffer, KeySize, Key, OffSetInDataFile any) {
	binary.Write(IndexContent, binary.LittleEndian, KeySize)          //ispisi duzinu kljuca(ovo je uvek readable jer je uint64)
	binary.Write(IndexContent, binary.LittleEndian, Key)              //ispisi kljuc
	binary.Write(IndexContent, binary.LittleEndian, OffSetInDataFile) //ispisi offset bloka u Data fajlu
}

func WriteToFile(generation int, level int, fileType string, fileOrganisation string, bufferToWrite *bytes.Buffer, TOCData *string) {
	err := ioutil.WriteFile("./Data/SSTables/"+fileOrganisation+"/"+fileType+"-"+strconv.Itoa(generation)+"-"+strconv.Itoa(level)+".bin", bufferToWrite.Bytes(), 0644)
	//adding paths of sstable files to TOC
	if err != nil {
		fmt.Println("Err u pisanju fajla "+fileType, err)
		return
	}
	*TOCData += "./Data/SSTables/" + fileOrganisation + "/" + fileType + "-" + strconv.Itoa(generation) + "-" + strconv.Itoa(level) + ".bin" + "\n"
}
func WriteToTxtFile(generation int, level int, fileType string, fileOrganisation string, data string, TOCData *string) {
	file, err := os.Create("./Data/SSTables/" + fileOrganisation + "/" + fileType + "-" + strconv.Itoa(generation) + "-" + strconv.Itoa(level) + ".txt")
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer file.Close()
	_, err = file.WriteString(data)
	if err != nil {
		fmt.Println("Error writing to file:", err)
		return
	}
	if TOCData != nil {
		//adding paths of sstable files to TOC
		*TOCData += "./Data/SSTables/" + fileOrganisation + "/" + fileType + "-" + strconv.Itoa(generation) + "-" + strconv.Itoa(level) + ".txt" + "\n"
	}
}

// SINGLE FILE
func BuildSSTableSingle(sortedLogs []*Log, generation, level, SUMMARY_BLOCK_SIZE int) error {
	header := Header{
		LogsOffset:    32,
		BloomOffset:   0,
		IndexOffset:   0,
		SummaryOffset: 0}

	// Serialize the logs to bytes
	var serializedLogs []byte
	for _, log := range sortedLogs {
		serializedLogs = append(serializedLogs, log.Serialize()...)
	}
	header.BloomOffset += header.LogsOffset + uint64(len(serializedLogs))

	// Build Bloom Filter
	filter := BuildFilter(sortedLogs, len(sortedLogs), 0.1)
	filterSerialized := filter.Serialize()
	header.IndexOffset += header.BloomOffset + uint64(filterSerialized.Len())

	// Build Index
	indexData := BuildIndex(sortedLogs, header.LogsOffset)
	serializedIndex := SerializeIndexes(indexData)

	// Build Summary
	summary := BuildSummary(indexData, header.IndexOffset, SUMMARY_BLOCK_SIZE)
	summarySerialized := summary.Bytes()
	header.SummaryOffset += header.IndexOffset + uint64(len(serializedIndex))

	var FileContent = new(bytes.Buffer)
	merkle := BuildMerkleTreeRoot(sortedLogs)
	binary.Write(FileContent, binary.LittleEndian, header.HeaderSerialize())
	binary.Write(FileContent, binary.LittleEndian, serializedLogs)
	binary.Write(FileContent, binary.LittleEndian, filterSerialized.Bytes())
	binary.Write(FileContent, binary.LittleEndian, serializedIndex)
	binary.Write(FileContent, binary.LittleEndian, summarySerialized)
	TOCData := ""
	WriteToFile(generation, level, "Data", "Single", FileContent, &TOCData)
	WriteToTxtFile(generation, level, "Metadata", "Single", hex.EncodeToString(SerializeMerkleTree(merkle)), &TOCData)
	WriteToTxtFile(generation, level, "TOC", "Single", TOCData, nil)
	return nil
}
