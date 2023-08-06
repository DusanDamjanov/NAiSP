package ConfigurationHandler

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

const configFilePath = "ConfigurationHandler/config.json"

type ConfigHandler struct {
	//WalSegmentSize --> velicina wal00001.log fajla
	NumOfWalSegmentLogs int     `json:"NumOfWalSegmentLogs"`
	MemtableStruct      string  `json:"MemtableStruct"`
	SizeOfMemtable      uint32  `json:"SizeOfMemtable"`
	Trashold            float64 `json:"Trashold"`
	NumOfFiles          string  `json:"NumOfFiles"`
	//if memtable struct is btree
	BTreeDegree uint32 `json:"BTreeDegree"`
	//else struct == skipList(onda mi trebaju elementi za skiplist kao sto za btree imam njegov degree
	CacheSize              int `json:"CacheSize"`
	TokenBucketSize        int `json:"TokenBucketSize"`
	TokenBucketRefreshTime int `json:"TokenBucketRefreshTime"`
}

func UseCustomConfiguration() *ConfigHandler {
	file, err := os.Open(configFilePath)
	if err != nil {
		fmt.Println("Err opening json file")
		return nil
	}
	defer file.Close()

	jsonData, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Println("Err reading json file")
		return nil
	}

	var config ConfigHandler
	err = json.Unmarshal(jsonData, &config)
	if err != nil {
		fmt.Println("Error unmarshaling json file")
		return nil
	}
	return &config
}

func UseDefaultConfiguration() *ConfigHandler {
	config := ConfigHandler{MemtableStruct: "btree", SizeOfMemtable: 30, Trashold: 0.7, BTreeDegree: 2, NumOfFiles: "multiple",
		TokenBucketSize: 3, TokenBucketRefreshTime: 10000, CacheSize: 4}
	return &config
}
