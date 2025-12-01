package typeconverter

import (
	"os/exec"
	"strings"

	"github.com/yejune/gotossr/internal/utils"

	_ "github.com/tkrajina/typescriptify-golang-structs/typescriptify"
)

// Start starts the type converter
// It gets the name of structs in PropsStructsPath and generates a temporary file to run the type converter
// structsFilePath can be comma-separated for multiple files
func Start(structsFilePath, generatedTypesPath string) error {
	// Split by comma for multiple files
	filePaths := strings.Split(structsFilePath, ",")

	var allStructNames []string
	var firstFilePath string

	for _, fp := range filePaths {
		fp = strings.TrimSpace(fp)
		if fp == "" {
			continue
		}
		if firstFilePath == "" {
			firstFilePath = fp
		}

		// Get struct names from file
		structNames, err := getStructNamesFromFile(fp)
		if err != nil {
			return err
		}
		allStructNames = append(allStructNames, structNames...)
	}

	if len(allStructNames) == 0 {
		return nil
	}

	// Create a folder for the temporary generator files
	cacheDir, err := utils.GetTypeConverterCacheDir()
	if err != nil {
		return err
	}

	// Create the generator file (using first file path for package resolution)
	temporaryFilePath, err := createTemporaryFileMulti(filePaths, generatedTypesPath, cacheDir, allStructNames)
	if err != nil {
		return err
	}

	// Run the file
	cmd := exec.Command("go", "run", temporaryFilePath)
	_, err = cmd.CombinedOutput()
	if err != nil {
		return err
	}
	return nil
}
