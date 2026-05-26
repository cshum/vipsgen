package generator

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// GenerationTask describes one template-to-file emission step.
type GenerationTask struct {
	TemplateFile string
	OutputFile   string
}

// GenerationPlan captures the ordered set of files that will be generated.
type GenerationPlan struct {
	Tasks []GenerationTask
}

// Generate generates all code files from templates by scanning the template directory
func Generate(
	templateLoader TemplateLoader,
	templateData *TemplateData,
	outputDir string,
) error {
	plan, err := BuildGenerationPlan(templateLoader, templateData, outputDir)
	if err != nil {
		return err
	}

	return executeGenerationPlan(templateLoader, templateData, plan)
}

// BuildGenerationPlan resolves the ordered set of template files to emit.
func BuildGenerationPlan(
	templateLoader TemplateLoader,
	templateData *TemplateData,
	outputDir string,
) (*GenerationPlan, error) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %v", err)
	}

	templateFiles, err := templateLoader.ListFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to list template files: %v", err)
	}

	plan := &GenerationPlan{}
	for _, templateFile := range templateFiles {
		baseName := filepath.Base(templateFile)
		if shouldSkipTemplate(baseName, templateData.IncludeTest) {
			log.Printf("Skipping test template: %s (use --include-test to generate)\n", baseName)
			continue
		}

		plan.Tasks = append(plan.Tasks, GenerationTask{
			TemplateFile: templateFile,
			OutputFile:   filepath.Join(outputDir, strings.TrimSuffix(baseName, ".tmpl")),
		})
	}

	return plan, nil
}

func executeGenerationPlan(
	templateLoader TemplateLoader,
	templateData *TemplateData,
	plan *GenerationPlan,
) error {
	var generatedFiles []string

	for _, task := range plan.Tasks {
		if err := templateLoader.GenerateFile(task.TemplateFile, task.OutputFile, templateData); err != nil {
			return fmt.Errorf("failed to generate %s: %v", task.OutputFile, err)
		}
		generatedFiles = append(generatedFiles, task.OutputFile)
	}

	log.Printf("\nSuccessfully generated files from templates: %d\n", len(generatedFiles))
	for _, file := range generatedFiles {
		log.Printf("  - %s\n", file)
	}

	return nil
}

func shouldSkipTemplate(templateFile string, includeTest bool) bool {
	if includeTest {
		return false
	}

	return strings.HasSuffix(templateFile, "_test.go.tmpl") ||
		strings.HasSuffix(templateFile, "_race.go.tmpl")
}
