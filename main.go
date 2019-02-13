package main

import (
	"encoding/json"
	"fmt"
	"github.com/cilium/cilium/test/helpers/constants"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-yaml/yaml"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	imageKey = "image"
)

var (

	// RootCmd represents the base command when called without any subcommands
	RootCmd = &cobra.Command{
		Use:   "image-list",
		Short: "list all validImages in YAML files in a given repository",
		Run: func(cmd *cobra.Command, args []string) {
		},
	}
	dirs          []string
	validateImages bool
	validImages   map[string]struct{}
	invalidImages map[string]struct{}

	validImagePrefixes = map[string]struct{}{
		"docker.io": {},
		"gcr.io": {},
		"k8s.gcr.io": {},
		"quay.io": {},
	}

	ignoreImagePrefixes = map[string]struct{}{
		"k8s1:5000": {},
	}
)

func init() {
	flags := RootCmd.Flags()
	flags.StringSliceVar(&dirs, "directories", []string{"."}, "Directories to search for validImages in YAML files")
	flags.BoolVarP(&validateImages, "validation", "v", false, "validate that images are from a specified container image host and are not using the \"latest\" tag")
	viper.BindPFlags(flags)
}

func main() {
	err := RootCmd.Execute()
	if err != nil {
		log.Fatal("error executing program")
	}

	validImages = make(map[string]struct{})
	for validImage := range constants.AllImages {
		validImages[validImage] = struct{}{}
	}
	invalidImages = make(map[string]struct{})

	for _, dir := range dirs {
		getImageNamesFromFilesInDirectory(dir, validImages, invalidImages)
	}

	if validateImages {
		fmt.Println("************* VALID IMAGES *************")
		printMapSortedByKeys(validImages)
		fmt.Println()
		fmt.Println("************* INVALID IMAGES *************")
		printMapSortedByKeys(invalidImages)
	} else {
		printMapSortedByKeys(validImages)
	}
}

func getImageNamesFromFilesInDirectory(dir string, validImages, invalidImages map[string]struct{}) {
	err := filepath.Walk(dir, func(fileName string, info os.FileInfo, err error) error {

		if strings.HasSuffix(fileName, "yaml") || strings.HasSuffix(fileName, "yml") {
			yamlFile, err := ioutil.ReadFile(fileName)
			if err != nil {
				fmt.Printf("ERROR READING YAML: %s\n", err)
				return nil
			}
			dec := yaml.NewDecoder(strings.NewReader(string(yamlFile)))
			for {
				var value interface{}
				err := dec.Decode(&value)
				if err == io.EOF {
					break
				}
				examineYAML("yaml", fileName, value, validImages, invalidImages)
			}
		} else if strings.HasSuffix(fileName, "json") {
			jsonFile, err := ioutil.ReadFile(fileName)
			if err != nil {
				fmt.Printf("ERROR READING JSON: %s\n", err)
				return nil
			}
			dec := json.NewDecoder(strings.NewReader(string(jsonFile)))
			for {
				var value interface{}
				err := dec.Decode(&value)
				if err == io.EOF {
					break
				}
				examineYAML("json", fileName, value, validImages, invalidImages)
			}

		}
		return nil
	})
	if err != nil {
		log.Fatalf("error reading files from directory: %s\n", err)
	}
}

func printMapSortedByKeys(strMap map[string]struct{}) {
	strList := make([]string, 0, len(strMap))
	for img := range strMap {
		strList = append(strList, img)
	}

	sort.Strings(strList)

	for _, img := range strList {
		fmt.Printf("%s\n", img)
	}

}

// Modified from: https://stackoverflow.com/questions/40737122/convert-yaml-to-json-without-struct#comment68705664_40737676
func examineYAML(kind, fileName string, i interface{}, validImages, invalidImages map[string]struct{}) {
	switch x := i.(type) {
	case map[interface{}]interface{}:
		for k, v := range x {
			switch k.(type) {
			case string:
				kStr := k.(string)
				if kStr == imageKey {
					vStr := v.(string)
					if validateImages {
						valid := processImage(vStr)
						if valid == nil {
							continue
						} else if *valid {
							validImages[vStr] = struct{}{}
						} else {
							invalidImages[vStr] = struct{}{}
						}
					} else {
						validImages[vStr] = struct{}{}
					}
				}
			}
			examineYAML(kind, fileName, v, validImages, invalidImages)
		}
	case []interface{}:
		for _, v := range x {
			examineYAML(kind, fileName, v, validImages, invalidImages)
		}
	case map[string]interface{}:
		for k, v := range x {
			if k == imageKey {
				vStr := v.(string)
				if validateImages {
					valid := processImage(vStr)
					if valid == nil {
						continue
					} else if *valid {
						validImages[vStr] = struct{}{}
					} else {
						invalidImages[vStr] = struct{}{}
					}
				} else {
					validImages[vStr] = struct{}{}
				}
			}
			examineYAML(kind, fileName, v, validImages, invalidImages)
		}
	}
}

func splitImageName(imgName string) []string {
	return strings.Split(imgName, "/")
}

func processImage(imgName string) *bool {

	falsePtr := false
	truePtr := true
	splitImg := splitImageName(imgName)
	splitImgColons := strings.Split(imgName, ":")

	if len(splitImgColons) > 0 {
		if splitImgColons[len(splitImgColons) - 1] == "latest" {
			return &falsePtr
		}
	} else {
		return &falsePtr
	}

	if len(splitImg) > 0 {
		if _, ok := ignoreImagePrefixes[splitImg[0]]; ok {
			return nil
		}
		if _, ok := validImagePrefixes[splitImg[0]]; ok {
			return &truePtr
		}
	}
	return &falsePtr
}
