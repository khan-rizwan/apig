package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/gedex/inflector"
)

const templateDir = "_templates"

var funcMap = template.FuncMap{
	"apibDefaultValue": apibDefaultValue,
	"apibType":         apibType,
	"pluralize":        inflector.Pluralize,
	"requestParams":    requestParams,
	"tolower":          strings.ToLower,
	"toSnakeCase":      toSnakeCase,
	"title":            strings.Title,
}

var skeletons = []string{
	"README.md.tmpl",
	".gitignore.tmpl",
	"main.go.tmpl",
	filepath.Join("db", "db.go.tmpl"),
	filepath.Join("db", "pagination.go.tmpl"),
	filepath.Join("router", "router.go.tmpl"),
	filepath.Join("middleware", "set_db.go.tmpl"),
	filepath.Join("server", "server.go.tmpl"),
	filepath.Join("helper", "field.go.tmpl"),
	filepath.Join("version", "version.go.tmpl"),
	filepath.Join("version", "version_test.go.tmpl"),
	filepath.Join("controllers", ".gitkeep.tmpl"),
	filepath.Join("docs", ".gitkeep.tmpl"),
	filepath.Join("models", ".gitkeep.tmpl"),
}

var managedFields = []string{
	"ID",
	"CreatedAt",
	"UpdatedAt",
}

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

// code from https://gist.github.com/stoewer/fbe273b711e6a06315d19552dd4d33e6
func toSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

func apibDefaultValue(field *Field) string {
	switch field.Type {
	case "bool":
		return "false"
	case "string":
		return strings.ToUpper(field.Name)
	case "time.Time":
		return "`2000-01-01 00:00:00`"
	case "*time.Time":
		return "`2000-01-01 00:00:00`"
	case "uint":
		return "1"
	}

	return ""
}

func apibType(field *Field) string {
	switch field.Type {
	case "bool":
		return "boolean"
	case "string":
		return "string"
	case "time.Time":
		return "string"
	case "*time.Time":
		return "string"
	case "uint":
		return "number"
	}

	switch field.Association.Type {
	case AssociationBelongsTo:
		return inflector.Pluralize(strings.ToLower(strings.Replace(field.Type, "*", "", -1)))
	case AssociationHasMany:
		return fmt.Sprintf("array[%s]", inflector.Pluralize(strings.ToLower(strings.Replace(field.Type, "[]", "", -1))))
	case AssociationHasOne:
		return inflector.Pluralize(strings.ToLower(strings.Replace(field.Type, "*", "", -1)))
	}

	return ""
}

func requestParams(fields []*Field) []*Field {
	var managed bool

	params := []*Field{}

	for _, field := range fields {
		managed = false

		for _, name := range managedFields {
			if field.Name == name {
				managed = true
			}
		}

		if !managed {
			params = append(params, field)
		}
	}

	return params
}

func generateApibIndex(detail *Detail, outDir string) error {
	body, err := Asset(filepath.Join(templateDir, "index.apib.tmpl"))

	if err != nil {
		return err
	}

	tmpl, err := template.New("apib").Funcs(funcMap).Parse(string(body))

	if err != nil {
		return err
	}

	var buf bytes.Buffer

	if err := tmpl.Execute(&buf, detail); err != nil {
		return err
	}

	dstPath := filepath.Join(outDir, "docs", "index.apib")

	if !fileExists(filepath.Dir(dstPath)) {
		if err := mkdir(filepath.Dir(dstPath)); err != nil {
			return err
		}
	}

	if err := ioutil.WriteFile(dstPath, buf.Bytes(), 0644); err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "\t\x1b[32m%s\x1b[0m %s\n", "create", dstPath)

	return nil
}

func generateApibModel(detail *Detail, outDir string) error {
	body, err := Asset(filepath.Join(templateDir, "model.apib.tmpl"))

	if err != nil {
		return err
	}

	tmpl, err := template.New("apib").Funcs(funcMap).Parse(string(body))

	if err != nil {
		return err
	}

	var buf bytes.Buffer

	if err := tmpl.Execute(&buf, detail); err != nil {
		return err
	}

	dstPath := filepath.Join(outDir, "docs", toSnakeCase(detail.Model.Name)+".apib")

	if !fileExists(filepath.Dir(dstPath)) {
		if err := mkdir(filepath.Dir(dstPath)); err != nil {
			return err
		}
	}

	if err := ioutil.WriteFile(dstPath, buf.Bytes(), 0644); err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "\t\x1b[32m%s\x1b[0m %s\n", "create", dstPath)

	return nil
}

func generateSkeleton(detail *Detail, outDir string) error {
	if fileExists(outDir) {
		fmt.Fprintf(os.Stderr, "%s is already exists", outDir)
		os.Exit(1)
	}

	for _, filename := range skeletons {
		srcPath := filepath.Join(templateDir, "skeleton", filename)
		dstPath := filepath.Join(outDir, strings.Replace(filename, ".tmpl", "", 1))

		body, err := Asset(srcPath)

		if err != nil {
			return err
		}

		tmpl, err := template.New("complex").Parse(string(body))

		if err != nil {
			return err
		}

		var buf bytes.Buffer

		if err := tmpl.Execute(&buf, detail); err != nil {
			return err
		}

		if !fileExists(filepath.Dir(dstPath)) {
			if err := mkdir(filepath.Dir(dstPath)); err != nil {
				return err
			}
		}

		if err := ioutil.WriteFile(dstPath, buf.Bytes(), 0644); err != nil {
			return err
		}

		fmt.Printf("\t\x1b[32m%s\x1b[0m %s\n", "create", dstPath)
	}

	return nil
}

func generateController(detail *Detail, outDir string) error {
	body, err := Asset(filepath.Join(templateDir, "controller.go.tmpl"))

	if err != nil {
		return err
	}

	tmpl, err := template.New("controller").Funcs(funcMap).Parse(string(body))

	if err != nil {
		return err
	}

	var buf bytes.Buffer

	if err := tmpl.Execute(&buf, detail); err != nil {
		return err
	}

	dstPath := filepath.Join(outDir, "controllers", toSnakeCase(detail.Model.Name)+".go")

	if !fileExists(filepath.Dir(dstPath)) {
		if err := mkdir(filepath.Dir(dstPath)); err != nil {
			return err
		}
	}

	if err := ioutil.WriteFile(dstPath, buf.Bytes(), 0644); err != nil {
		return err
	}

	fmt.Printf("\t\x1b[32m%s\x1b[0m %s\n", "create", dstPath)

	return nil
}

func generateREADME(models []*Model, outDir string) error {
	body, err := Asset(filepath.Join(templateDir, "README.md.tmpl"))

	if err != nil {
		return err
	}

	tmpl, err := template.New("readme").Funcs(funcMap).Parse(string(body))

	if err != nil {
		return err
	}

	var buf bytes.Buffer

	if err := tmpl.Execute(&buf, models); err != nil {
		return err
	}

	dstPath := filepath.Join(outDir, "README.md")

	if !fileExists(filepath.Dir(dstPath)) {
		if err := mkdir(filepath.Dir(dstPath)); err != nil {
			return err
		}
	}

	if err := ioutil.WriteFile(dstPath, buf.Bytes(), 0644); err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "\t\x1b[32m%s\x1b[0m %s\n", "update", dstPath)

	return nil
}

func generateRouter(detail *Detail, outDir string) error {
	body, err := Asset(filepath.Join(templateDir, "router.go.tmpl"))

	if err != nil {
		return err
	}

	tmpl, err := template.New("router").Funcs(funcMap).Parse(string(body))

	if err != nil {
		return err
	}

	var buf bytes.Buffer

	if err := tmpl.Execute(&buf, detail); err != nil {
		return err
	}

	dstPath := filepath.Join(outDir, "router", "router.go")

	if !fileExists(filepath.Dir(dstPath)) {
		if err := mkdir(filepath.Dir(dstPath)); err != nil {
			return err
		}
	}

	if err := ioutil.WriteFile(dstPath, buf.Bytes(), 0644); err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "\t\x1b[32m%s\x1b[0m %s\n", "update", dstPath)

	return nil
}

func generateDB(detail *Detail, outDir string) error {
	body, err := Asset(filepath.Join(templateDir, "db.go.tmpl"))

	if err != nil {
		return err
	}

	tmpl, err := template.New("db").Funcs(funcMap).Parse(string(body))

	if err != nil {
		return err
	}

	var buf bytes.Buffer

	if err := tmpl.Execute(&buf, detail); err != nil {
		return err
	}

	dstPath := filepath.Join(outDir, "db", "db.go")

	if !fileExists(filepath.Dir(dstPath)) {
		if err := mkdir(filepath.Dir(dstPath)); err != nil {
			return err
		}
	}

	if err := ioutil.WriteFile(dstPath, buf.Bytes(), 0644); err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "\t\x1b[32m%s\x1b[0m %s\n", "update", dstPath)

	return nil
}
