package generator

import (
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"text/template"
)

type stubTemplateLoader struct {
	listFilesErr error
	listFiles    []string
	generated    []GenerationTask
	generateErr  error
	generateAt   int
}

func (s *stubTemplateLoader) LoadTemplate(name string) (*template.Template, error) {
	return nil, nil
}

func (s *stubTemplateLoader) ListFiles() ([]string, error) {
	if s.listFilesErr != nil {
		return nil, s.listFilesErr
	}
	return append([]string(nil), s.listFiles...), nil
}

func (s *stubTemplateLoader) GenerateFile(templateName, outputFile string, data interface{}) error {
	s.generated = append(s.generated, GenerationTask{TemplateFile: templateName, OutputFile: outputFile})
	if s.generateErr != nil && len(s.generated) == s.generateAt {
		return s.generateErr
	}
	return nil
}

func TestBuildGenerationPlanSkipsTestsWhenDisabled(t *testing.T) {
	loader := &stubTemplateLoader{
		listFiles: []string{"a.go.tmpl", "b_test.go.tmpl", "nested/c.go.tmpl"},
	}
	outputDir := t.TempDir()

	plan, err := BuildGenerationPlan(loader, &TemplateData{}, outputDir)
	if err != nil {
		t.Fatalf("BuildGenerationPlan returned error: %v", err)
	}

	got := plan.Tasks
	want := []GenerationTask{
		{TemplateFile: "a.go.tmpl", OutputFile: filepath.Join(outputDir, "a.go")},
		{TemplateFile: "nested/c.go.tmpl", OutputFile: filepath.Join(outputDir, "c.go")},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected plan tasks\n got: %#v\nwant: %#v", got, want)
	}
}

func TestBuildGenerationPlanIncludesTestsWhenEnabled(t *testing.T) {
	loader := &stubTemplateLoader{
		listFiles: []string{"a.go.tmpl", "b_test.go.tmpl"},
	}
	outputDir := t.TempDir()

	plan, err := BuildGenerationPlan(loader, &TemplateData{IncludeTest: true}, outputDir)
	if err != nil {
		t.Fatalf("BuildGenerationPlan returned error: %v", err)
	}

	got := []string{plan.Tasks[0].OutputFile, plan.Tasks[1].OutputFile}
	want := []string{filepath.Join(outputDir, "a.go"), filepath.Join(outputDir, "b_test.go")}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected output files\n got: %#v\nwant: %#v", got, want)
	}
}

func TestGeneratePreservesExecutionOrder(t *testing.T) {
	loader := &stubTemplateLoader{
		listFiles: []string{"z.go.tmpl", "a.go.tmpl", "b_test.go.tmpl"},
	}

	outputDir := t.TempDir()
	if err := Generate(loader, &TemplateData{}, outputDir); err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	got := loader.generated
	want := []GenerationTask{
		{TemplateFile: "z.go.tmpl", OutputFile: filepath.Join(outputDir, "z.go")},
		{TemplateFile: "a.go.tmpl", OutputFile: filepath.Join(outputDir, "a.go")},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected generation order\n got: %#v\nwant: %#v", got, want)
	}
}

func TestGenerateWrapsFileGenerationErrors(t *testing.T) {
	loader := &stubTemplateLoader{
		listFiles:   []string{"a.go.tmpl", "b.go.tmpl"},
		generateErr: errors.New("boom"),
		generateAt:  2,
	}
	outputDir := t.TempDir()

	err := Generate(loader, &TemplateData{}, outputDir)
	if err == nil {
		t.Fatal("Generate returned nil error")
	}
	if !strings.Contains(err.Error(), filepath.Join(outputDir, "b.go")) {
		t.Fatalf("expected error to mention output file, got: %v", err)
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected wrapped cause, got: %v", err)
	}
}