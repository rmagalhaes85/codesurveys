package importer

import (
	"database/sql"
	"errors"
	"fmt"
	_ "io/fs"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	_ "github.com/lib/pq"
	"gopkg.in/yaml.v3"
)

type Snippet struct {
	Language string
	OriginalFilename string
	Category string
	Content string
}

type SnippetTuple struct {
	Language string
	OriginalPath string
	Snippets []*Snippet
}

type Experiment struct {
	Categories []string `yaml:categories`
	SnippetTuples []*SnippetTuple
}

func (exp Experiment) CategoriesMatch(categories []string) bool {
	if len(exp.Categories) != len(categories) {
		return false
	}
	for _, cat := range(categories) {
		if !slices.Contains(exp.Categories, cat) {
			return false
		}
	}
	return true
}

func ImportSnippets(dir string) error {
	connStr := "postgres://elsendi:fusquinha@localhost:5432/readability?sslmode=disable"

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to open database connection: %v", err)
		return err
	}
	log.Print("Database connection established")
	defer db.Close()

	var version string
	err = db.QueryRow("select version()").Scan(&version)
	if err != nil {
		log.Fatalf("Query failed: %v", err)
		return err
	}

	// ensure database is not initialized with a set of snippets
	err = checkDbNotInitialized(db)
	if err != nil {
		log.Fatalf("Error validating database: %v", err)
		return err
	}

	//experiment1 := &Experiment{
	//	Categories: []string{"a", "c"},
	//}
	//fmt.Printf("Categories match? %v\n", experiment1.CategoriesMatch([]string{"a", "b"}))
	//var experiment *Experiment
	//if experiment == nil {
	//	fmt.Println("Experiment ainda eh nil")
	//	return nil
	//}

	// determine snippets directory
	err = validateSnippetsDirectory(dir)
	if err != nil {
		log.Fatalf("Can't access snippets directory %s: %v", dir, err)
		return err
	}


	return nil
}

func readExperiment(experimentFilePath string) (*Experiment, error) {
	buf, err := ioutil.ReadFile(experimentFilePath)
	if err != nil {
		return nil, fmt.Errorf("Error reading experiment descriptor file: %w", err)
	}

	experiment := &Experiment{}
	err = yaml.Unmarshal(buf, experiment)
	if err != nil {
		return nil, fmt.Errorf("Error unmarshalling experiment yaml: %w", err)
	}

	return experiment, nil
}

func checkDbNotInitialized(db *sql.DB) error {
	var workingSetCount int
	err := db.QueryRow("select count(*) from working_sets").Scan(&workingSetCount)

	if err != nil {
		return errors.New("Query to check if database is initialized failed")
	}
	if workingSetCount > 0 {
		return errors.New("Database is already initialized")
	}

	return nil
}

func validateSnippetsDirectory(dir string) error {
	const experimentFilename = "experiment.yaml"
	const snippetDescriptorFilename = "snippets.yaml"

	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("The directory %s doesn't exist", dir)
		}
	}
	if !info.IsDir() {
		return fmt.Errorf("The path %s is not a directory", dir)
	}

	experimentFilePath := filepath.Join(dir, experimentFilename)
	info, err = os.Stat(experimentFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("The experiment descriptor experiment.yaml file doesn't exist")
		}
		return fmt.Errorf("Error checking experiment.yaml file: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("The file %s exists but it's a directory", experimentFilePath)
	}

	var experiment *Experiment
	experiment, err = readExperiment(experimentFilePath)
	if err != nil {
		return err
	}

	log.Printf("Experiment read: %w", experiment)

	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, file := range files {
		if file.IsDir() {
			var descriptorFilename = filepath.Join(dir, file.Name(), snippetDescriptorFilename)
			log.Printf("Checking file %s", descriptorFilename)
			_, err := os.Stat(descriptorFilename)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return fmt.Errorf("Error checking for %s file", snippetDescriptorFilename)
			}

			snippetsDir, err := os.ReadDir(filepath.Join(dir, file.Name()))
			if err != nil {
				return fmt.Errorf("Error opening directory %v", file.Name())
			}
			log.Print("Found a directory with snippets")

			var listOfCategories []string
			for _, snippetFile := range snippetsDir {
				fileWithoutExt := strings.TrimSuffix(snippetFile.Name(), path.Ext(snippetFile.Name()))
				if fileWithoutExt == "snippets" {
					continue
				}
				listOfCategories = append(listOfCategories, fileWithoutExt)
			}

			if !experiment.CategoriesMatch(listOfCategories) {
				return fmt.Errorf("Categories in the experiment descriptor don't match with files in the %s directory. Categories: %v",
					filepath.Join(dir, file.Name()), listOfCategories)
			}
		}
	}

	//var experiment Experiment
	//var firstSnippetTuple = true

	//err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
	//	if err != nil {
	//		return err
	//	}

	//	if info, err = os.Stat(filepath.Join(path, snippetDescriptorFilename)); err != nil {
	//		if os.IsNotExist(err) {
	//			return filepath.SkipDir
	//		}
	//		return err
	//	}

	//	return nil
	//})

	//if err != nil {
	//	return err
	//}

	return nil
}


	// for the first tuple of snippets

	// - determine how many snippets are there in the tuple. There must be more than
	// one.

	// - determine the category represented by each snippet in the tuple. This
	// category is named after the base file name of the snippet file. There should be
	// an extension in each snippet file too -- the language and therefore the syntax
	// highlighting rules will be selected after it.

	// for each snippet tuple

	// - ensure that the number of snippets is the same as the first snippets tuple

	// - ensure that the snippets category names correspond exactly to the categories
	// in the first snippets tuple

	// - ensure no snippet in the tuple is empty

	// - determine snippet language from the file extension

	// - generate an HTML syntax-highlighted version of the snippet

	// - store it in the database
