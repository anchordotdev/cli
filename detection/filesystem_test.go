package detection

import (
	"os"
	"reflect"
	"slices"
	"testing"
	"testing/fstest"

	"github.com/anchordotdev/cli/anchorcli"
)

func TestFileDetector_Detect(t *testing.T) {
	testCases := []struct {
		name string

		detector FileDetector
		fs       FS

		match Match
		err   error
	}{
		{
			name: "Mock Exact Match",

			detector: FileDetector{
				Title:          "Test Detector",
				Paths:          []string{"1.txt", "2.txt", "3.txt", "4.txt", "5.txt"},
				AnchorCategory: anchorcli.CategoryCustom,
			},
			fs: fstest.MapFS{
				"1.txt": emptyFile,
				"2.txt": emptyFile,
				"3.txt": emptyFile,
				"4.txt": emptyFile,
				"5.txt": emptyFile,
			},

			match: Match{
				Detected:       true,
				Confidence:     High,
				AnchorCategory: anchorcli.CategoryCustom,
			},
		},
		{
			name: "Mock Strong Match",

			detector: FileDetector{
				Title: "Test Detector",
				Paths: []string{"1.txt", "2.txt", "3.txt", "4.txt", "5.txt"},
			},
			fs: fstest.MapFS{
				"1.txt": emptyFile,
				"2.txt": emptyFile,
				"3.txt": emptyFile,
			},

			match: Match{
				Detected:       true,
				Confidence:     Medium,
				AnchorCategory: anchorcli.CategoryCustom,
			},
		},
		{
			name: "Mock Possible Match",

			detector: FileDetector{
				Title: "Test Detector",
				Paths: []string{"1.txt", "2.txt", "3.txt", "4.txt", "5.txt"},
			},
			fs: fstest.MapFS{
				"1.txt": emptyFile,
				"2.txt": emptyFile,
			},

			match: Match{
				Detected:       true,
				Confidence:     Low,
				AnchorCategory: anchorcli.CategoryCustom,
			},
		},
		{
			name: "Mock None Match",

			detector: FileDetector{
				Title: "Test Detector",
				Paths: []string{"1.txt", "2.txt", "3.txt", "4.txt", "5.txt"},
			},
			fs: fstest.MapFS{},

			match: Match{
				Detected:       false,
				Confidence:     None,
				AnchorCategory: anchorcli.CategoryCustom,
			},
		},
		{
			name: "Missing RequiredFiles forces match to Low",

			detector: FileDetector{
				Title:         "Test Detector",
				Paths:         []string{"1.txt", "2.txt", "3.txt", "4.txt", "5.txt"},
				RequiredFiles: []string{"1.txt"},
			},
			fs: fstest.MapFS{
				"2.txt": emptyFile,
				"3.txt": emptyFile,
				"4.txt": emptyFile,
				"5.txt": emptyFile,
			},

			match: Match{
				Detected:             true,
				Confidence:           Low,
				AnchorCategory:       anchorcli.CategoryCustom,
				MissingRequiredFiles: []string{"1.txt"},
			},
		},
		{
			name: "Missing Required Files never forces None match to Low",

			detector: FileDetector{
				Title:         "Test Detector",
				Paths:         []string{"20.txt", "40.txt", "60.txt", "80.txt", "100.txt"},
				RequiredFiles: []string{"1.txt"},
			},
			fs: fstest.MapFS{
				"20.txt": emptyFile,
			},

			match: Match{
				Detected:             true,
				Confidence:           None,
				AnchorCategory:       anchorcli.CategoryCustom,
				MissingRequiredFiles: []string{"1.txt"},
			},
		},
	}

	for _, test := range testCases {
		test := test

		t.Run(test.name, func(t *testing.T) {
			match, err := test.detector.Detect(test.fs)
			if err != nil {
				if want, got := test.err, err; want != got {
					t.Fatalf("want error %q, but got %q", want, got)
				}
			}

			if want, got := test.match.Detected, match.Detected; want != got {
				t.Errorf("want match detection %t, got %t", want, got)
			}
			if want, got := test.match.Confidence, match.Confidence; want != got {
				t.Errorf("want match confidence score %s, got %s", want, got)
			}
			if want, got := len(test.match.FollowUpDetectors), len(match.FollowUpDetectors); want != got {
				t.Errorf("want %d follow-up detectors, got %d", want, got)
			}
			if want, got := test.match.FollowUpDetectors, match.FollowUpDetectors; !reflect.DeepEqual(want, got) {
				t.Errorf("want %+v follow-up detectors, got %+v", want, got)
			}
			if want, got := test.match.AnchorCategory, match.AnchorCategory; want != got {
				t.Errorf("want AnchorCategory %s, got %s", want, got)
			}
			if want, got := test.match.MissingRequiredFiles, match.MissingRequiredFiles; slices.Compare(want, got) != 0 {
				t.Errorf("want missing required files %+v, got %+v", want, got)
			}
		})
	}
}

func TestFileDetector_DetectWithoutMockFS(t *testing.T) {
	// Specify the name of an existing directory that contains the expected files
	directory, err := os.Getwd()
	if err != nil {
		t.Fatalf("Unexpected Getwd error: %v", err)
	}
	dirFS := os.DirFS(directory).(FS)

	files, err := os.ReadDir(directory)
	if err != nil {
		t.Fatalf("Unexpected ReadDir error: %v", err)
	}

	// Choose a random file from working directory
	expectedFiles := []string{files[0].Name()}

	// Create the FileDetector with the expected file paths
	// Instantiating one without a FileSystem will use the actual OS File System
	detector := FileDetector{
		Title: "Test Detector",
		Paths: expectedFiles,
	}

	// Perform the detection
	match, err := detector.Detect(dirFS)

	if err != nil {
		t.Fatalf("Unexpected error during detection: %v", err)
	}

	// Verify the detection result
	if !match.Detected {
		t.Errorf("Expected detection result to be true, but got false")
	}

	if match.Confidence != High {
		t.Errorf("Expected confidence to be High, but got %s", match.Confidence)
	}
}

func TestFileDetector_GetTitle(t *testing.T) {
	detector := FileDetector{
		Title: "Test Detector",
		Paths: []string{
			"file1.txt",
			"file2.txt",
		},
	}

	expectedTitle := "Test Detector"
	actualTitle := detector.GetTitle()

	if actualTitle != expectedTitle {
		t.Errorf("Expected detector name %s, but got %s", expectedTitle, actualTitle)
	}
}

func TestFileDetector_FollowUp(t *testing.T) {
	// Create a FileDetector with some follow-up detectors
	detector1 := FileDetector{Title: "Detector 1", Paths: []string{"file1.txt"}}
	detector2 := FileDetector{Title: "Detector 2", Paths: []string{"file2.txt"}}
	detector3 := FileDetector{Title: "Detector 3", Paths: []string{"file3.txt"}}

	parentDetector := FileDetector{
		Title:             "Parent Detector",
		Paths:             []string{"parent.txt"},
		FollowUpDetectors: []Detector{&detector1, &detector2, &detector3},
	}

	// Get the follow-up detectors
	followUpDetectors := parentDetector.FollowUp()

	// Verify the expected follow-up detectors
	expectedFollowUpDetectors := []Detector{&detector1, &detector2, &detector3}
	if len(followUpDetectors) != len(expectedFollowUpDetectors) {
		t.Errorf("Expected %d follow-up detectors, but got %d", len(expectedFollowUpDetectors), len(followUpDetectors))
		return
	}

	for i, expectedDetector := range expectedFollowUpDetectors {
		actualDetector := followUpDetectors[i]
		if actualDetector.GetTitle() != expectedDetector.GetTitle() {
			t.Errorf("Expected follow-up detector name %s, but got %s", expectedDetector.GetTitle(), actualDetector.GetTitle())
		}
	}
}
