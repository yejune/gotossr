package typeconverter

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/yejune/gotossr/internal/utils"
)

// https://github.com/tkrajina/typescriptify-golang-structs/blob/master/tscriptify/main.go#L139
func createTemporaryFile(structsFilePath, generatedTypesPath, cacheDir string, structNames []string) (string, error) {
	return createTemporaryFileMulti([]string{structsFilePath}, generatedTypesPath, cacheDir, structNames)
}

// createTemporaryFileMulti creates a temporary file for multiple struct files
func createTemporaryFileMulti(structsFilePaths []string, generatedTypesPath, cacheDir string, structNames []string) (string, error) {
	temporaryFilePath := filepath.ToSlash(filepath.Join(cacheDir, "generator.go"))
	file, err := os.Create(temporaryFilePath)
	if err != nil {
		return temporaryFilePath, err
	}
	defer file.Close()

	t := template.Must(template.New("").Parse(TEMPLATE))

	// Collect unique module names and their aliases
	moduleAliases := make(map[string]string) // moduleName -> alias
	aliasCounter := 0

	for _, fp := range structsFilePaths {
		fp = strings.TrimSpace(fp)
		if fp == "" {
			continue
		}
		moduleName, err := getModuleName(fp)
		if err != nil {
			return temporaryFilePath, err
		}
		if _, exists := moduleAliases[moduleName]; !exists {
			alias := fmt.Sprintf("m%d", aliasCounter)
			if aliasCounter == 0 {
				alias = "m" // first module uses 'm' for backward compatibility
			}
			moduleAliases[moduleName] = alias
			aliasCounter++
		}
	}

	// Build imports
	var imports []string
	for moduleName, alias := range moduleAliases {
		imports = append(imports, fmt.Sprintf(`%s "%s"`, alias, moduleName))
	}

	// Build struct references with correct aliases
	structsArr := make([]string, 0)
	moduleForStruct := make(map[string]string) // structName -> moduleName

	for _, fp := range structsFilePaths {
		fp = strings.TrimSpace(fp)
		if fp == "" {
			continue
		}
		moduleName, _ := getModuleName(fp)
		fileStructs, _ := getStructNamesFromFile(fp)
		for _, s := range fileStructs {
			moduleForStruct[s] = moduleName
		}
	}

	for _, structName := range structNames {
		structName = strings.TrimSpace(structName)
		if len(structName) > 0 {
			moduleName := moduleForStruct[structName]
			alias := moduleAliases[moduleName]
			structsArr = append(structsArr, alias+"."+structName)
		}
	}

	var params TemplateParams
	params.Imports = imports
	params.Structs = structsArr
	params.Interface = true
	params.TargetFile = utils.GetFullFilePath(generatedTypesPath)

	err = t.Execute(file, params)
	if err != nil {
		return temporaryFilePath, err
	}

	return temporaryFilePath, nil
}

// getModuleName gets the module name of the props structs file
func getModuleName(propsStructsPath string) (string, error) {
	dir := filepath.ToSlash(filepath.Dir(utils.GetFullFilePath(propsStructsPath)))
	cmd := exec.Command("go", "list")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}
