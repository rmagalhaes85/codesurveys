package importer

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	_ "github.com/lib/pq"
	"github.com/alecthomas/chroma/v2/quick"
	"gopkg.in/yaml.v3"
)

const connStr = "postgres://elsendi:fusquinha@localhost:5432/readability?sslmode=disable"

type snippet struct {
	Language string
	OriginalFilepath string
	Category string
	Content string
}

type snippetTuple struct {
	Language string
	OriginalPath string
	Snippets []*snippet
	Categories []string
}

type experiment struct {
	Language string
	Categories []string `yaml:categories`
	SnippetTuples []*snippetTuple
}

func ImportExperiment(experimentDir string) (err error) {
	log.Print("Importing experiment...")

	var db *sql.DB
	db, err = connectAndValidateDb()
	if err != nil {
		return
	}
	defer db.Close()

	var exp *experiment
	exp, err = loadExperimentFromDisk(experimentDir)
	if err != nil {
		return
	}

	err = storeExperiment(db, exp)

	log.Print("The experiment was successfully imported")
	return
}

func connectAndValidateDb() (db *sql.DB, err error) {
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		return
	}

	var initialized bool
	initialized, err = isDbInitialized(db)
	if err != nil {
		return nil, err
	}
	if initialized {
		return nil, errors.New("Database is already initialized")
	}

	return
}

func isDbInitialized(db *sql.DB) (bool, error) {
	var workingSetCount int
	err := db.QueryRow("select count(*) from working_sets").Scan(&workingSetCount)

	if err != nil {
		return false, errors.New("Query to check if database is initialized failed")
	}

	return (workingSetCount > 0), nil
}

func loadExperimentFromDisk(experimentDir string) (exp *experiment, err error) {
	const snippetTupleDescriptorFilename = "snippets.yaml"

	exp, err = parseExperimentIfValid(experimentDir)
	if err != nil {
		return
	}

	var experimentFiles []os.DirEntry
	experimentFiles, err = os.ReadDir(experimentDir)
	if err != nil {
		return
	}

	for _, experimentFile := range experimentFiles {
		path := filepath.Join(experimentDir, experimentFile.Name())
		snippetTuple, err := parseSnippetTupleIfValid(path, exp)
		if err != nil {
			return nil, err
		}
		if snippetTuple == nil {
			continue
		}
		fmt.Println("snippetTuple = %v", snippetTuple)
		exp.SnippetTuples = append(exp.SnippetTuples, snippetTuple)
	}

	fmt.Printf("len(exp.SnippetTuples) = %v\n", len(exp.SnippetTuples))
	fmt.Printf("exp.SnippetTuples[0] = %v\n", exp.SnippetTuples[0])
	//fmt.Printf("len(exp.SnippetTuples[0].Snippets) = %v\n", len(exp.SnippetTuples[0].Snippets))

	return
}

func parseExperimentIfValid(experimentDir string) (exp *experiment, err error) {
	const experimentDescriptorFilename = "experiment.yaml"
	info, err := os.Stat(experimentDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("The directory %s doesn't exist", experimentDir)
		}
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("The path %s is not a directory", experimentDir)
	}

	experimentDescriptorPath := filepath.Join(experimentDir, experimentDescriptorFilename)
	info, err = os.Stat(experimentDescriptorPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("The experiment descriptor experiment.yaml file doesn't exist")
		}
		return nil, fmt.Errorf("Error checking experiment.yaml file: %w", err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("The file %s exists but it's a directory", experimentDescriptorPath)
	}

	buf, err := ioutil.ReadFile(experimentDescriptorPath)
	if err != nil {
		return nil, fmt.Errorf("Error reading experiment descriptor file: %w", err)
	}
	exp = &experiment{}
	err = yaml.Unmarshal(buf, exp)
	if err != nil {
		return nil, fmt.Errorf("Error unmarshalling exp yaml: %w", err)
	}

	fmt.Printf("experimentDescriptorPath = %s\n", experimentDescriptorPath)
	fmt.Printf("exp = %v\n", exp)
	fmt.Printf("buf = %v\n", string(buf))
	err = experimentIsValid(exp)
	if err != nil {
		return nil, err
	}

	return exp, nil
}

func experimentIsValid(exp *experiment) error {
	if len(exp.Categories) < 2 {
		return fmt.Errorf("Expected more than 1 category in the experiment. Found only %d", len(exp.Categories))
	}
	return nil
}

func parseSnippetTupleIfValid(path string, exp *experiment) (st *snippetTuple, err error) {
	// -- caso o subdiretório tenha um arquivo snippets.yaml
	// -- carregar um objeto snippetTuple com os dados de snippets.yaml
	// -- para cada arquivo nesse subdiretório
	// --- carregar os dados do arquivo e gerar um snippet formatado
	// --- armazenar esses dados num objeto snippet
	// --- adicionar esse snippet a snippetTuple-mae
	// -- validar que o snippetTuple tem snippets correspondendo eatamente às
	// categorias informadas no experiment.yaml

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("Directory %s doesn't exist", path)
		}
		return nil, err
	}
	if !info.IsDir() {
		// this function has probably received a path for a FILE in the experiment
		// folder. Here, we're only interested in subfolders
		return nil, nil
	}

	const snippetDescriptorFilename = "snippets.yaml"
	var snippetDescriptorPath = filepath.Join(path, snippetDescriptorFilename)
	info, err = os.Stat(snippetDescriptorPath)
	if err != nil {
		if os.IsNotExist(err) {
			// simply ignore the directory in case there's no snippets.yaml file
			return nil, nil
		}
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("The snippet descriptor file exists, but is a directory (%s)", snippetDescriptorPath)
	}

	var snippetTupleFiles []os.DirEntry
	snippetTupleFiles, err = os.ReadDir(path)
	if err != nil {
		return
	}

	st = &snippetTuple{}
	// if the <path> directory contains a snippet descriptor file (snippets.yaml), the
	// system assumes every other files in the folder to be source code snippets files
	for _, stFile := range snippetTupleFiles {
		if stFile.Name() == snippetDescriptorFilename {
			continue
		}
		var stFileCategory = getFilenameWithoutExt(stFile.Name())
		var s *snippet
		s = &snippet{
			Category: stFileCategory,
			OriginalFilepath: filepath.Join(path, stFile.Name()),
		}
		st.Categories = append(st.Categories, stFileCategory)
		st.Snippets = append(st.Snippets, s)
	}

	if !categoriesMatch(exp.Categories, st.Categories) {
		// TODO make error message more informative
		err = errors.New("Categories don't match")
		return
	}

	// we leave the actual snippet's contents loading for the end, after all the
	// snippet tuples have been validated
	err = loadSnippetsContents(st, exp)
	if err != nil {
		return
	}

	return
}

func getFilenameWithoutExt(originalName string) string {
	return strings.TrimSuffix(originalName, path.Ext(originalName))
}

func loadSnippetsContents(st *snippetTuple, exp *experiment) error {
	var buf bytes.Buffer
	for _, s := range st.Snippets {
		content, err := os.ReadFile(s.OriginalFilepath)
		if err != nil {
			return err
		}
		language := determineLanguage(s, st, exp)
		err = quick.Highlight(&buf, string(content), language, "html", "monokai")
		if err != nil {
			return err
		}
		s.Content = buf.String()
	}

	return nil
}

func determineLanguage(s *snippet, st *snippetTuple, exp *experiment) string {
	snippetLanguage := strings.TrimSpace(s.Language)
	stLanguage := strings.TrimSpace(st.Language)
	expLanguage := strings.TrimSpace(exp.Language)

	if len(snippetLanguage) > 0 {
		return snippetLanguage
	}
	if len(stLanguage) > 0 {
		return stLanguage
	}
	if len(expLanguage) > 0 {
		return expLanguage
	}
	// TODO try to determine language based on file extension
	// (path.Ext(s.OriginalFilepath))
	return ""
}

func categoriesMatch(cat1 []string, cat2 []string) bool {
	if cat1 == nil || cat2 == nil {
		return false
	}
	if len(cat1) != len(cat2) {
		return false
	}
	for _, cat := range(cat1) {
		if !slices.Contains(cat2, cat) {
			return false
		}
	}
	return true
}

func storeExperiment(db *sql.DB, exp *experiment) (err error) {
	_ = db
	_ = exp
	return nil
}
