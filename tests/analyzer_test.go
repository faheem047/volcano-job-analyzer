package tests

import (
    "testing"
    "time"

    "volcano-job-analyzer/pkg/analyzer"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    volcanov1alpha1 "volcano.sh/apis/pkg/apis/batch/v1alpha1"
    corev1 "k8s.io/api/core/v1"
    "k8s.io/apimachinery/pkg/api/resource"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAnalyzer_AnalyzeJob(t *testing.T) {
    tests := []struct {
        name           string
        job            *volcanov1alpha1.Job
        expectedTasks  int
        expectedJobType analyzer.JobType
        expectError    bool
    }{
        {
            name:           "PyTorch Job Analysis",
            job:            createPyTorchJob(),
            expectedTasks:  2,
            expectedJobType: analyzer.JobTypePyTorch,
            expectError:    false,
        },
        {
            name:           "TensorFlow Job Analysis",
            job:            createTensorFlowJob(),
            expectedTasks:  3,
            expectedJobType: analyzer.JobTypeTensorFlow,
            expectError:    false,
        },
        {
            name:           "Empty Job",
            job:            createEmptyJob(),
            expectedTasks:  0,
            expectedJobType: analyzer.JobTypeGeneric,
            expectError:    false,
        },
    }

    config := &analyzer.AnalyzerConfig{
        EnableAdvancedMetrics:       true,
        NetworkLatencyThreshold:     10 * time.Millisecond,
        ResourceEfficiencyThreshold: 0.8,
        DefaultJobTimeout:           2 * time.Hour,
    }

    a := analyzer.NewAnalyzer(config)

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := a.AnalyzeJob(tt.job)

            if tt.expectError {
                assert.Error(t, err)
                return
            }

            require.NoError(t, err)
            assert.Equal(t, tt.expectedTasks, len(result.Tasks))
            assert.Equal(t, tt.expectedJobType, result.JobType)
            assert.Equal(t, tt.job.Name, result.JobName)
            assert.Equal(t, tt.job.Namespace, result.Namespace)
        })
    }
}

func TestAnalyzer_ResourceCalculation(t *testing.T) {
    job := createResourceIntensiveJob()
    config := &analyzer.AnalyzerConfig{
        EnableAdvancedMetrics:       true,
        NetworkLatencyThreshold:     10 * time.Millisecond,
        ResourceEfficiencyThreshold: 0.8,
        DefaultJobTimeout:           2 * time.Hour,
    }

    a := analyzer.NewAnalyzer(config)
    result, err := a.AnalyzeJob(job)

    require.NoError(t, err)

    // Verify total resource calculation
    expectedCPU := resource.MustParse("12000m") // 4000m * 1 + 2000m * 4
    expectedMemory := resource.MustParse("24Gi") // 8Gi * 1 + 4Gi * 4
    expectedGPU := resource.MustParse("4")       // 0 * 1 + 1 * 4

    assert.True(t, result.TotalResources.CPU.Equal(expectedCPU), 
        "Expected CPU: %s, Got: %s", expectedCPU.String(), result.TotalResources.CPU.String())
    assert.True(t, result.TotalResources.Memory.Equal(expectedMemory),
        "Expected Memory: %s, Got: %s", expectedMemory.String(), result.TotalResources.Memory.String())
    assert.True(t, result.TotalResources.GPU.Equal(expectedGPU),
        "Expected GPU: %s, Got: %s", expectedGPU.String(), result.TotalResources.GPU.String())
}

func TestAnalyzer_TaskClassification(t *testing.T) {
    tests := []struct {
        taskName     string
        expectedType analyzer.TaskType
    }{
        {"master", analyzer.TaskTypeMaster},
        {"worker", analyzer.TaskTypeWorker},
        {"parameter-server", analyzer.TaskTypeParameterServer},
        {"ps", analyzer.TaskTypeParameterServer},
        {"driver", analyzer.TaskTypeDriver},
        {"executor", analyzer.TaskTypeExecutor},
        {"scheduler", analyzer.TaskTypeScheduler},
        {"unknown-task", analyzer.TaskTypeUnknown},
    }

    config := &analyzer.AnalyzerConfig{
        EnableAdvancedMetrics:       true,
        NetworkLatencyThreshold:     10 * time.Millisecond,
        ResourceEfficiencyThreshold: 0.8,
        DefaultJobTimeout:           2 * time.Hour,
    }

    a := analyzer.NewAnalyzer(config)

    for _, tt := range tests {
        t.Run(tt.taskName, func(t *testing.T) {
            job := createJobWithTaskName(tt.taskName)
            result, err := a.AnalyzeJob(job)

            require.NoError(t, err)
            assert.Len(t, result.Tasks, 1)
            assert.Equal(t, tt.expectedType, result.Tasks[0].TaskType)
        })
    }
}

func TestAnalyzer_ScalabilityMetrics(t *testing.T) {
    job := createScalableJob()
    config := &analyzer.AnalyzerConfig{
        EnableAdvancedMetrics:       true,
        NetworkLatencyThreshold:     10 * time.Millisecond,
        ResourceEfficiencyThreshold: 0.8,
        DefaultJobTimeout:           2 * time.Hour,
    }

    a := analyzer.NewAnalyzer(config)
    result, err := a.AnalyzeJob(job)

    require.NoError(t, err)

    metrics := result.ScalabilityMetrics
    assert.GreaterOrEqual(t, metrics.MinimumNodes, int32(1))
    assert.GreaterOrEqual(t, metrics.OptimalNodes, metrics.MinimumNodes)
    assert.GreaterOrEqual(t, metrics.MaximumNodes, metrics.OptimalNodes)
    assert.Greater(t, metrics.ScalingFactor, float64(1))
}

// Helper functions to create test jobs

func createPyTorchJob() *volcanov1alpha1.Job {
    return &volcanov1alpha1.Job{
        ObjectMeta: metav1.ObjectMeta{
            Name:      "pytorch-test",
            Namespace: "default",
        },
        Spec: volcanov1alpha1.JobSpec{
            Tasks: []volcanov1alpha1.TaskSpec{
                {
                    Name:     "master",
                    Replicas: 1,
                    Template: corev1.PodTemplateSpec{
                        Spec: corev1.PodSpec{
                            Containers: []corev1.Container{
                                {
                                    Name:  "pytorch",
                                    Image: "pytorch/pytorch:latest",
                                    Resources: corev1.ResourceRequirements{
                                        Requests: corev1.ResourceList{
                                            corev1.ResourceCPU:    resource.MustParse("2000m"),
                                            corev1.ResourceMemory: resource.MustParse("4Gi"),
                                        },
                                    },
                                },
                            },
                        },
                    },
                },
                {
                    Name:     "worker",
                    Replicas: 2,
                    Template: corev1.PodTemplateSpec{
                        Spec: corev1.PodSpec{
                            Containers: []corev1.Container{
                                {
                                    Name:  "pytorch",
                                    Image: "pytorch/pytorch:latest",
                                    Resources: corev1.ResourceRequirements{
                                        Requests: corev1.ResourceList{
                                            corev1.ResourceCPU:    resource.MustParse("1000m"),
                                            corev1.ResourceMemory: resource.MustParse("2Gi"),
                                            "nvidia.com/gpu":      resource.MustParse("1"),
                                        },
                                    },
                                },
                            },
                        },
                    },
                },
            },
        },
    }
}

func createTensorFlowJob() *volcanov1alpha1.Job {
    return &volcanov1alpha1.Job{
        ObjectMeta: metav1.ObjectMeta{
            Name:      "tensorflow-test",
            Namespace: "default",
        },
        Spec: volcanov1alpha1.JobSpec{
            Tasks: []volcanov1alpha1.TaskSpec{
                {
                    Name:     "chief",
                    Replicas: 1,
                    Template: corev1.PodTemplateSpec{
                        Spec: corev1.PodSpec{
                            Containers: []corev1.Container{
                                {
                                    Name:  "tensorflow",
                                    Image: "tensorflow/tensorflow:latest",
                                    Resources: corev1.ResourceRequirements{
                                        Requests: corev1.ResourceList{
                                            corev1.ResourceCPU:    resource.MustParse("4000m"),
                                            corev1.ResourceMemory: resource.MustParse("8Gi"),
                                        },
                                    },
                                },
                            },
                        },
                    },
                },
                {
                    Name:     "worker",
                    Replicas: 2,
                    Template: corev1.PodTemplateSpec{
                        Spec: corev1.PodSpec{
                            Containers: []corev1.Container{
                                {
                                    Name:  "tensorflow",
                                    Image: "tensorflow/tensorflow:latest",
                                    Resources: corev1.ResourceRequirements{
                                        Requests: corev1.ResourceList{
                                            corev1.ResourceCPU:    resource.MustParse("2000m"),
                                            corev1.ResourceMemory: resource.MustParse("4Gi"),
                                        },
                                    },
                                },
                            },
                        },
                    },
                },
                {
                    Name:     "ps",
                    Replicas: 1,
                    Template: corev1.PodTemplateSpec{
                        Spec: corev1.PodSpec{
                            Containers: []corev1.Container{
                                {
                                    Name:  "tensorflow",
                                    Image: "tensorflow/tensorflow:latest",
                                    Resources: corev1.ResourceRequirements{
                                        Requests: corev1.ResourceList{
                                            corev1.ResourceCPU:    resource.MustParse("1000m"),
                                            corev1.ResourceMemory: resource.MustParse("2Gi"),
                                        },
                                    },
                                },
                            },
                        },
                    },
                },
            },
        },
    }
}

func createEmptyJob() *volcanov1alpha1.Job {
    return &volcanov1alpha1.Job{
        ObjectMeta: metav1.ObjectMeta{
            Name:      "empty-test",
            Namespace: "default",
        },
        Spec: volcanov1alpha1.JobSpec{
            Tasks: []volcanov1alpha1.TaskSpec{},
        },
    }
}

func createResourceIntensiveJob() *volcanov1alpha1.Job {
    return &volcanov1alpha1.Job{
        ObjectMeta: metav1.ObjectMeta{
            Name:      "resource-intensive-test",
            Namespace: "default",
        },
        Spec: volcanov1alpha1.JobSpec{
            Tasks: []volcanov1alpha1.TaskSpec{
                {
                    Name:     "master",
                    Replicas: 1,
                    Template: corev1.PodTemplateSpec{
                        Spec: corev1.PodSpec{
                            Containers: []corev1.Container{
                                {
                                    Name:  "master",
                                    Image: "test:latest",
                                    Resources: corev1.ResourceRequirements{
                                        Requests: corev1.ResourceList{
                                            corev1.ResourceCPU:    resource.MustParse("4000m"),
                                            corev1.ResourceMemory: resource.MustParse("8Gi"),
                                        },
                                    },
                                },
                            },
                        },
                    },
                },
                {
                    Name:     "worker",
                    Replicas: 4,
                    Template: corev1.PodTemplateSpec{
                        Spec: corev1.PodSpec{
                            Containers: []corev1.Container{
                                {
                                    Name:  "worker",
                                    Image: "test:latest",
                                    Resources: corev1.ResourceRequirements{
                                        Requests: corev1.ResourceList{
                                            corev1.ResourceCPU:    resource.MustParse("2000m"),
                                            corev1.ResourceMemory: resource.MustParse("4Gi"),
                                            "nvidia.com/gpu":      resource.MustParse("1"),
                                        },
                                    },
                                },
                            },
                        },
                    },
                },
            },
        },
    }
}

func createJobWithTaskName(taskName string) *volcanov1alpha1.Job {
    return &volcanov1alpha1.Job{
        ObjectMeta: metav1.ObjectMeta{
            Name:      "test-job",
            Namespace: "default",
        },
        Spec: volcanov1alpha1.JobSpec{
            Tasks: []volcanov1alpha1.TaskSpec{
                {
                    Name:     taskName,
                    Replicas: 1,
                    Template: corev1.PodTemplateSpec{
                        Spec: corev1.PodSpec{
                            Containers: []corev1.Container{
                                {
                                    Name:  "test",
                                    Image: "test:latest",
                                    Resources: corev1.ResourceRequirements{
                                        Requests: corev1.ResourceList{
                                            corev1.ResourceCPU:    resource.MustParse("1000m"),
                                            corev1.ResourceMemory: resource.MustParse("1Gi"),
                                        },
                                    },
                                },
                            },
                        },
                    },
                },
            },
        },
    }
}

func createScalableJob() *volcanov1alpha1.Job {
    return &volcanov1alpha1.Job{
        ObjectMeta: metav1.ObjectMeta{
            Name:      "scalable-test",
            Namespace: "default",
        },
        Spec: volcanov1alpha1.JobSpec{
            Tasks: []volcanov1alpha1.TaskSpec{
                {
                    Name:     "master",
                    Replicas: 1,
                    Template: corev1.PodTemplateSpec{
                        Spec: corev1.PodSpec{
                            Containers: []corev1.Container{
                                {
                                    Name:  "master",
                                    Image: "test:latest",
                                    Resources: corev1.ResourceRequirements{
                                        Requests: corev1.ResourceList{
                                            corev1.ResourceCPU:    resource.MustParse("1000m"),
                                            corev1.ResourceMemory: resource.MustParse("2Gi"),
                                        },
                                    },
                                },
                            },
                        },
                    },
                },
                {
                    Name:     "worker",
                    Replicas: 8,
                    Template: corev1.PodTemplateSpec{
                        Spec: corev1.PodSpec{
                            Containers: []corev1.Container{
                                {
                                    Name:  "worker",
                                    Image: "test:latest",
                                    Resources: corev1.ResourceRequirements{
                                        Requests: corev1.ResourceList{
                                            corev1.ResourceCPU:    resource.MustParse("500m"),
                                            corev1.ResourceMemory: resource.MustParse("1Gi"),
                                        },
                                    },
                                },
                            },
                        },
                    },
                },
            },
        },
    }
}