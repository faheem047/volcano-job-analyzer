package parser

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    
    volcanov1alpha1 "volcano.sh/apis/pkg/apis/batch/v1alpha1"
    "sigs.k8s.io/yaml"
    "k8s.io/apimachinery/pkg/util/validation"
)

// Parser handles Volcano Job parsing and validation
type Parser struct {
    validator *Validator
}

// NewParser creates a new parser instance
func NewParser() *Parser {
    return &Parser{
        validator: NewValidator(),
    }
}

// ParseVolcanoJobFromFile reads and parses a Volcano Job from YAML file
func ParseVolcanoJobFromFile(filename string) (*volcanov1alpha1.Job, error) {
    data, err := ioutil.ReadFile(filename)
    if err != nil {
        return nil, fmt.Errorf("failed to read file %s: %w", filename, err)
    }

    return ParseVolcanoJobFromYAML(data)
}

// ParseVolcanoJobFromYAML parses Volcano Job from YAML bytes
func ParseVolcanoJobFromYAML(data []byte) (*volcanov1alpha1.Job, error) {
    job := &volcanov1alpha1.Job{}
    
    if err := yaml.Unmarshal(data, job); err != nil {
        return nil, fmt.Errorf("failed to unmarshal YAML: %w", err)
    }

    // Basic validation
    if job.APIVersion == "" {
        job.APIVersion = "batch.volcano.sh/v1alpha1"
    }
    
    if job.Kind == "" {
        job.Kind = "Job"
    }

    // Validate the parsed job
    if err := validateBasicStructure(job); err != nil {
        return nil, fmt.Errorf("invalid job structure: %w", err)
    }

    return job, nil
}

// validateBasicStructure performs basic structural validation
func validateBasicStructure(job *volcanov1alpha1.Job) error {
    if job.Name == "" {
        return fmt.Errorf("job name is required")
    }

    if errs := validation.IsDNS1123Label(job.Name); len(errs) > 0 {
        return fmt.Errorf("invalid job name: %v", errs)
    }

    if len(job.Spec.Tasks) == 0 {
        return fmt.Errorf("job must have at least one task")
    }

    for i, task := range job.Spec.Tasks {
        if task.Name == "" {
            return fmt.Errorf("task[%d] name is required", i)
        }

        if task.Replicas <= 0 {
            return fmt.Errorf("task[%d] must have replicas > 0", i)
        }

        if len(task.Template.Spec.Containers) == 0 {
            return fmt.Errorf("task[%d] must have at least one container", i)
        }

        for j, container := range task.Template.Spec.Containers {
            if container.Name == "" {
                return fmt.Errorf("task[%d].container[%d] name is required", i, j)
            }

            if container.Image == "" {
                return fmt.Errorf("task[%d].container[%d] image is required", i, j)
            }
        }
    }

    return nil
}

// ParseMultipleJobsFromFile parses multiple Volcano Jobs from a single YAML file
func ParseMultipleJobsFromFile(filename string) ([]*volcanov1alpha1.Job, error) {
    data, err := ioutil.ReadFile(filename)
    if err != nil {
        return nil, fmt.Errorf("failed to read file %s: %w", filename, err)
    }

    return ParseMultipleJobsFromYAML(data)
}

// ParseMultipleJobsFromYAML parses multiple Volcano Jobs from YAML bytes
func ParseMultipleJobsFromYAML(data []byte) ([]*volcanov1alpha1.Job, error) {
    // Split YAML documents
    docs := splitYAMLDocuments(data)
    jobs := make([]*volcanov1alpha1.Job, 0, len(docs))

    for i, doc := range docs {
        if len(doc) == 0 {
            continue
        }

        job, err := ParseVolcanoJobFromYAML(doc)
        if err != nil {
            return nil, fmt.Errorf("failed to parse document %d: %w", i, err)
        }

        jobs = append(jobs, job)
    }

    if len(jobs) == 0 {
        return nil, fmt.Errorf("no valid Volcano Jobs found")
    }

    return jobs, nil
}

// splitYAMLDocuments splits YAML content by document separator
func splitYAMLDocuments(data []byte) [][]byte {
    // Simple implementation - split by "---"
    // In production, you might want to use a more robust YAML splitter
    content := string(data)
    docs := [][]byte{}
    
    // Split by document separator
    parts := []string{content} // Simplified for this example
    
    for _, part := range parts {
        if trimmed := []byte(part); len(trimmed) > 0 {
            docs = append(docs, trimmed)
        }
    }

    return docs
}

// ExtractJobMetadata extracts metadata from a Volcano Job without full parsing
func ExtractJobMetadata(data []byte) (*JobMetadata, error) {
    // Use a minimal structure to extract just metadata
    var metadata struct {
        APIVersion string `yaml:"apiVersion"`
        Kind       string `yaml:"kind"`
        Metadata   struct {
            Name        string            `yaml:"name"`
            Namespace   string            `yaml:"namespace"`
            Labels      map[string]string `yaml:"labels"`
            Annotations map[string]string `yaml:"annotations"`
        } `yaml:"metadata"`
    }

    if err := yaml.Unmarshal(data, &metadata); err != nil {
        return nil, fmt.Errorf("failed to extract metadata: %w", err)
    }

    return &JobMetadata{
        Name:        metadata.Metadata.Name,
        Namespace:   metadata.Metadata.Namespace,
        Labels:      metadata.Metadata.Labels,
        Annotations: metadata.Metadata.Annotations,
        APIVersion:  metadata.APIVersion,
        Kind:        metadata.Kind,
    }, nil
}

// JobMetadata represents extracted job metadata
type JobMetadata struct {
    Name        string            `json:"name"`
    Namespace   string            `json:"namespace"`
    Labels      map[string]string `json:"labels"`
    Annotations map[string]string `json:"annotations"`
    APIVersion  string            `json:"apiVersion"`
    Kind        string            `json:"kind"`
}

// ConvertToYAML converts a Volcano Job back to YAML
func ConvertToYAML(job *volcanov1alpha1.Job) ([]byte, error) {
    return yaml.Marshal(job)
}

// ConvertToJSON converts a Volcano Job to JSON
func ConvertToJSON(job *volcanov1alpha1.Job) ([]byte, error) {
    return json.Marshal(job)
}