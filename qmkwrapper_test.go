package qmkwrapper

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/leep-frog/command/command"
	"github.com/leep-frog/command/commander"
	"github.com/leep-frog/command/commandertest"
	"github.com/leep-frog/command/commandtest"
)

type readFileResponse struct {
	expectedFile string
	contents     string
	err          error
}

type writeFileResponse struct {
	expectedFile string
	expectedData string
	err          error
}

func TestMain(t *testing.T) {
	// Don't want a shared object because it can change in each test run.
	qw := func() *qmkWrapper {
		return &qmkWrapper{
			QMKDir:    filepath.Join("initial", "qmk", "dir"),
			OutputDir: filepath.Join("initial", "output", "dir"),
		}
	}
	qwHash := func(hash1, hash2 string) *qmkWrapper {
		return &qmkWrapper{
			QMKDir:    filepath.Join("initial", "qmk", "dir"),
			OutputDir: filepath.Join("initial", "output", "dir"),
			hash:      hash1,
			hash2:     hash2,
		}
	}
	for _, test := range []struct {
		name               string
		q                  *qmkWrapper
		want               *qmkWrapper
		readFileResponses  []*readFileResponse
		writeFileResponses []*writeFileResponse
		etc                *commandtest.ExecuteTestCase
	}{
		{
			name: "fails if qmk dir isn't set",
			q: &qmkWrapper{
				OutputDir: qw().OutputDir,
			},
			etc: &commandtest.ExecuteTestCase{
				Args: []string{
					"kb",
					"km",
					"--codes",
					"message 1",
					"message two",
				},
				WantStderr: "Directory values have not been set (`q config set`)\n",
				WantErr:    fmt.Errorf("Directory values have not been set (`q config set`)"),
			},
		},
		{
			name: "fails if output dir isn't set",
			q: &qmkWrapper{
				QMKDir: qw().QMKDir,
			},
			etc: &commandtest.ExecuteTestCase{
				Args: []string{
					"kb",
					"km",
					"--codes",
					"message 1",
					"message two",
				},
				WantStderr: "Directory values have not been set (`q config set`)\n",
				WantErr:    fmt.Errorf("Directory values have not been set (`q config set`)"),
			},
		},
		{
			name: "fails if can't get version",
			q:    qw(),
			etc: &commandtest.ExecuteTestCase{
				Args: []string{
					"kb",
					"km",
					"--codes",
					"message 1",
					"message two",
				},
				WantRunContents: []*commandtest.RunContents{{
					Name: "git",
					Args: []string{"rev-parse", "HEAD"},
					Dir:  qw().QMKDir,
				}},
				RunResponses: []*commandtest.FakeRun{{
					Err: fmt.Errorf("version oops"),
				}},
				WantData: &command.Data{Values: map[string]interface{}{
					keyboardArg.Name(): "kb",
					keymapArg.Name():   "km",
					codesFlag.Name():   []string{"message 1", "message two"},
					hexFileFlag.Name(): "bin",
				}},
				WantStderr: "failed to execute shell command: version oops\n",
				WantErr:    fmt.Errorf("failed to execute shell command: version oops"),
			},
		},
		{
			name: "fails if can't write to file",
			q:    qw(),
			writeFileResponses: []*writeFileResponse{
				{
					expectedFile: filepath.Join(qw().QMKDir, codeFile),
					expectedData: strings.Join([]string{
						"#pragma once",
						`#define LEEP_VERSION "abc123"`,
						`#define LEEP_CODE_1 "message 1"`,
						`#define LEEP_CODE_2 "message two"`,
						"",
					}, "\n"),
					err: fmt.Errorf("whoops"),
				},
			},
			etc: &commandtest.ExecuteTestCase{
				Args: []string{
					"kb",
					"km",
					"--codes",
					"message 1",
					"message two",
				},
				WantRunContents: []*commandtest.RunContents{{
					Name: "git",
					Args: []string{"rev-parse", "HEAD"},
					Dir:  qw().QMKDir,
				}},
				RunResponses: []*commandtest.FakeRun{{
					Stdout: []string{"abc123def456"},
				}},
				WantData: &command.Data{Values: map[string]interface{}{
					keyboardArg.Name(): "kb",
					keymapArg.Name():   "km",
					codesFlag.Name():   []string{"message 1", "message two"},
					hexFileFlag.Name(): "bin",
					"VERSION":          "abc123def456",
				}},
				WantStderr: "failed to write code file: whoops\n",
				WantErr:    fmt.Errorf("failed to write code file: whoops"),
			},
		},
		{
			name: "fails if qmk failure",
			q:    qw(),
			writeFileResponses: []*writeFileResponse{
				// Write codes to file
				{
					expectedFile: filepath.Join(qw().QMKDir, codeFile),
					expectedData: strings.Join([]string{
						"#pragma once",
						`#define LEEP_VERSION "abc123"`,
						`#define LEEP_CODE_1 "message 1"`,
						`#define LEEP_CODE_2 "message two"`,
						"",
					}, "\n"),
				},
				// Write empty strings to file
				{
					expectedFile: filepath.Join(qw().QMKDir, codeFile),
					expectedData: strings.Join([]string{
						"#pragma once",
						`#define LEEP_VERSION "auto-generated"`,
						`#define LEEP_CODE_1 ""`,
						`#define LEEP_CODE_2 ""`,
						"",
					}, "\n"),
				},
			},
			etc: &commandtest.ExecuteTestCase{
				Args: []string{
					"kb",
					"km",
					"--codes",
					"message 1",
					"message two",
				},
				RunResponses: []*commandtest.FakeRun{
					{
						Stdout: []string{"abc123def456"},
					},
					{
						Stdout: []string{"so"},
						Stderr: []string{"se"},
						Err:    fmt.Errorf("oops"),
					},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					keyboardArg.Name(): "kb",
					keymapArg.Name():   "km",
					codesFlag.Name():   []string{"message 1", "message two"},
					hexFileFlag.Name(): "bin",
					"VERSION":          "abc123def456",
				}},
				WantRunContents: []*commandtest.RunContents{
					{
						Name: "git",
						Args: []string{"rev-parse", "HEAD"},
						Dir:  qw().QMKDir,
					},
					{
						Name: "qmk",
						Args: []string{
							"compile",
							"--keyboard", "kb",
							"--keymap", "km",
						},
					},
				},
				WantStdout: "so\n",
				WantStderr: strings.Join([]string{
					"se",
					"failed to run qmk compile: failed to execute shell command: oops",
					"",
				}, "\n"),
				WantErr: fmt.Errorf("failed to run qmk compile: failed to execute shell command: oops"),
			},
		},
		{
			name: "fails if qmk failure + re-write failure",
			q:    qw(),
			writeFileResponses: []*writeFileResponse{
				// Write codes to file
				{
					expectedFile: filepath.Join(qw().QMKDir, codeFile),
					expectedData: strings.Join([]string{
						"#pragma once",
						`#define LEEP_VERSION "abc123"`,
						`#define LEEP_CODE_1 "message 1"`,
						`#define LEEP_CODE_2 "message two"`,
						"",
					}, "\n"),
				},
				// Write empty strings to file
				{
					expectedFile: filepath.Join(qw().QMKDir, codeFile),
					expectedData: strings.Join([]string{
						"#pragma once",
						`#define LEEP_VERSION "auto-generated"`,
						`#define LEEP_CODE_1 ""`,
						`#define LEEP_CODE_2 ""`,
						"",
					}, "\n"),
					err: fmt.Errorf("re-write goof"),
				},
			},
			etc: &commandtest.ExecuteTestCase{
				Args: []string{
					"kb",
					"km",
					"--codes",
					"message 1",
					"message two",
				},
				RunResponses: []*commandtest.FakeRun{
					{
						Stdout: []string{"abc123def456"},
					},
					{
						Stdout: []string{"so"},
						Stderr: []string{"se"},
						Err:    fmt.Errorf("oops"),
					},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					keyboardArg.Name(): "kb",
					keymapArg.Name():   "km",
					codesFlag.Name():   []string{"message 1", "message two"},
					hexFileFlag.Name(): "bin",
					"VERSION":          "abc123def456",
				}},
				WantRunContents: []*commandtest.RunContents{
					{
						Name: "git",
						Args: []string{"rev-parse", "HEAD"},
						Dir:  qw().QMKDir,
					},
					{
						Name: "qmk",
						Args: []string{
							"compile",
							"--keyboard", "kb",
							"--keymap", "km",
						},
					},
				},
				WantStdout: "so\n",
				WantStderr: strings.Join([]string{
					"se",
					"failed to run qmk compile: failed to execute shell command: oops",
					"CRITICAL: failed to remove temporary codes: re-write goof",
					"",
				}, "\n"),
				WantErr: fmt.Errorf("failed to run qmk compile: failed to execute shell command: oops"),
			},
		},
		{
			name: "fails if copy read failure",
			q:    qw(),
			writeFileResponses: []*writeFileResponse{
				// Write codes to file
				{
					expectedFile: filepath.Join(qw().QMKDir, codeFile),
					expectedData: strings.Join([]string{
						"#pragma once",
						`#define LEEP_VERSION "abc123"`,
						`#define LEEP_CODE_1 "message 1"`,
						`#define LEEP_CODE_2 "message two"`,
						"",
					}, "\n"),
				},
				// Write empty strings to file
				{
					expectedFile: filepath.Join(qw().QMKDir, codeFile),
					expectedData: strings.Join([]string{
						"#pragma once",
						`#define LEEP_VERSION "auto-generated"`,
						`#define LEEP_CODE_1 ""`,
						`#define LEEP_CODE_2 ""`,
						"",
					}, "\n"),
				},
			},
			readFileResponses: []*readFileResponse{{
				// Copy file read
				expectedFile: filepath.Join(qw().QMKDir, "kb_km.bin"),
				contents:     "abcd",
				err:          fmt.Errorf("rats"),
			}},
			etc: &commandtest.ExecuteTestCase{
				Args: []string{
					"kb",
					"km",
					"--codes",
					"message 1",
					"message two",
				},
				RunResponses: []*commandtest.FakeRun{
					{
						Stdout: []string{"abc123"},
					},
					{
						Stdout: []string{"so"},
						Stderr: []string{"se"},
					},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					keyboardArg.Name(): "kb",
					keymapArg.Name():   "km",
					codesFlag.Name():   []string{"message 1", "message two"},
					hexFileFlag.Name(): "bin",
					"VERSION":          "abc123",
				}},
				WantRunContents: []*commandtest.RunContents{
					{
						Name: "git",
						Args: []string{"rev-parse", "HEAD"},
						Dir:  qw().QMKDir,
					},
					{
						Name: "qmk",
						Args: []string{
							"compile",
							"--keyboard", "kb",
							"--keymap", "km",
						},
					},
				},
				WantStdout: "so\n",
				WantStderr: strings.Join([]string{
					"se",
					"failed to copy qmk files: failed to read input file: rats",
					"",
				}, "\n"),
				WantErr: fmt.Errorf("failed to copy qmk files: failed to read input file: rats"),
			},
		},
		{
			name: "fails if copy write failure",
			q:    qw(),
			writeFileResponses: []*writeFileResponse{
				// Write codes to file
				{
					expectedFile: filepath.Join(qw().QMKDir, codeFile),
					expectedData: strings.Join([]string{
						"#pragma once",
						`#define LEEP_VERSION "abc12"`,
						`#define LEEP_CODE_1 "message 1"`,
						`#define LEEP_CODE_2 "message two"`,
						"",
					}, "\n"),
				},
				// Copy write
				{
					expectedFile: filepath.Join(qw().OutputDir, "kb_km.bin"),
					expectedData: "abcd",
					err:          fmt.Errorf("argh"),
				},
				// Write empty strings to file
				{
					expectedFile: filepath.Join(qw().QMKDir, codeFile),
					expectedData: strings.Join([]string{
						"#pragma once",
						`#define LEEP_VERSION "auto-generated"`,
						`#define LEEP_CODE_1 ""`,
						`#define LEEP_CODE_2 ""`,
						"",
					}, "\n"),
				},
			},
			readFileResponses: []*readFileResponse{{
				// Copy file read
				expectedFile: filepath.Join(qw().QMKDir, "kb_km.bin"),
				contents:     "abcd",
			}},
			etc: &commandtest.ExecuteTestCase{
				Args: []string{
					"kb",
					"km",
					"--codes",
					"message 1",
					"message two",
				},
				RunResponses: []*commandtest.FakeRun{
					{
						Stdout: []string{"abc12"},
					},
					{
						Stdout: []string{"so"},
						Stderr: []string{"se"},
					},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					keyboardArg.Name(): "kb",
					keymapArg.Name():   "km",
					codesFlag.Name():   []string{"message 1", "message two"},
					hexFileFlag.Name(): "bin",
					"VERSION":          "abc12",
				}},
				WantRunContents: []*commandtest.RunContents{
					{
						Name: "git",
						Args: []string{"rev-parse", "HEAD"},
						Dir:  qw().QMKDir,
					},
					{
						Name: "qmk",
						Args: []string{
							"compile",
							"--keyboard", "kb",
							"--keymap", "km",
						},
					},
				},
				WantStdout: "so\n",
				WantStderr: strings.Join([]string{
					"se",
					"failed to copy qmk files: failed to write to output file: argh",
					"",
				}, "\n"),
				WantErr: fmt.Errorf("failed to copy qmk files: failed to write to output file: argh"),
			},
		},
		{
			name: "succeeds",
			q:    qw(),
			writeFileResponses: []*writeFileResponse{
				// Write codes to file
				{
					expectedFile: filepath.Join(qw().QMKDir, codeFile),
					expectedData: strings.Join([]string{
						"#pragma once",
						`#define LEEP_VERSION "abc123"`,
						`#define LEEP_CODE_1 "message 1"`,
						`#define LEEP_CODE_2 "message two"`,
						"",
					}, "\n"),
				},
				// Copy write
				{
					expectedFile: filepath.Join(qw().OutputDir, "kb_km.bin"),
					expectedData: "abcd",
				},
				// Write empty strings to file
				{
					expectedFile: filepath.Join(qw().QMKDir, codeFile),
					expectedData: strings.Join([]string{
						"#pragma once",
						`#define LEEP_VERSION "auto-generated"`,
						`#define LEEP_CODE_1 ""`,
						`#define LEEP_CODE_2 ""`,
						"",
					}, "\n"),
				},
			},
			readFileResponses: []*readFileResponse{{
				// Copy file read
				expectedFile: filepath.Join(qw().QMKDir, "kb_km.bin"),
				contents:     "abcd",
			}},
			etc: &commandtest.ExecuteTestCase{
				Args: []string{
					"kb",
					"km",
					"--codes",
					"message 1",
					"message two",
				},
				RunResponses: []*commandtest.FakeRun{
					{
						Stdout: []string{"abc123def456"},
					},
					{
						Stdout: []string{"so"},
						Stderr: []string{"se"},
					},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					keyboardArg.Name(): "kb",
					keymapArg.Name():   "km",
					codesFlag.Name():   []string{"message 1", "message two"},
					hexFileFlag.Name(): "bin",
					"VERSION":          "abc123def456",
				}},
				WantRunContents: []*commandtest.RunContents{
					{
						Name: "git",
						Args: []string{"rev-parse", "HEAD"},
						Dir:  qw().QMKDir,
					},
					{
						Name: "qmk",
						Args: []string{
							"compile",
							"--keyboard", "kb",
							"--keymap", "km",
						},
					},
				},
				WantStdout: "so\n",
				WantStderr: "se\n",
			},
		},
		{
			name: "succeeds with multiple keyboard/keymap parts and hex file flag",
			q:    qw(),
			writeFileResponses: []*writeFileResponse{
				// Write codes to file
				{
					expectedFile: filepath.Join(qw().QMKDir, codeFile),
					expectedData: strings.Join([]string{
						"#pragma once",
						`#define LEEP_VERSION "abc123"`,
						`#define LEEP_CODE_1 "message 1"`,
						`#define LEEP_CODE_2 "message two"`,
						"",
					}, "\n"),
				},
				// Copy write
				{
					expectedFile: filepath.Join(qw().OutputDir, "kb_sub_thing_km_more_path.hex"),
					expectedData: "abcd",
				},
				// Write empty strings to file
				{
					expectedFile: filepath.Join(qw().QMKDir, codeFile),
					expectedData: strings.Join([]string{
						"#pragma once",
						`#define LEEP_VERSION "auto-generated"`,
						`#define LEEP_CODE_1 ""`,
						`#define LEEP_CODE_2 ""`,
						"",
					}, "\n"),
				},
			},
			readFileResponses: []*readFileResponse{{
				// Copy file read
				expectedFile: filepath.Join(qw().QMKDir, "kb_sub_thing_km_more_path.hex"),
				contents:     "abcd",
			}},
			etc: &commandtest.ExecuteTestCase{
				Args: []string{
					"kb/sub\\thing",
					"km\\more/path",
					"--codes",
					"message 1",
					"message two",
					"-x",
				},
				RunResponses: []*commandtest.FakeRun{
					{
						Stdout: []string{"abc123def456"},
					},
					{
						Stdout: []string{"so"},
						Stderr: []string{"se"},
					},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					keyboardArg.Name(): "kb/sub\\thing",
					keymapArg.Name():   "km\\more/path",
					codesFlag.Name():   []string{"message 1", "message two"},
					hexFileFlag.Name(): "hex",
					"VERSION":          "abc123def456",
				}},
				WantRunContents: []*commandtest.RunContents{
					{
						Name: "git",
						Args: []string{"rev-parse", "HEAD"},
						Dir:  qw().QMKDir,
					},
					{
						Name: "qmk",
						Args: []string{
							"compile",
							"--keyboard", "kb/sub\\thing",
							"--keymap", "km\\more/path",
						},
					},
				},
				WantStdout: "so\n",
				WantStderr: "se\n",
			},
		},
		{
			name: "succeeds with noop rot (maxRuneChar)",
			q:    qwHash("abcd", "1234"),
			writeFileResponses: []*writeFileResponse{
				// Write codes to file
				{
					expectedFile: filepath.Join(qw().QMKDir, codeFile),
					expectedData: strings.Join([]string{
						"#pragma once",
						`#define LEEP_VERSION "abc"`,
						`#define LEEP_CODE_1 "abcd"`,
						`#define LEEP_CODE_2 "1234"`,
						"",
					}, "\n"),
				},
				// Copy write
				{
					expectedFile: filepath.Join(qw().OutputDir, "kb_km.bin"),
					expectedData: "abcd",
				},
				// Write empty strings to file
				{
					expectedFile: filepath.Join(qw().QMKDir, codeFile),
					expectedData: strings.Join([]string{
						"#pragma once",
						`#define LEEP_VERSION "auto-generated"`,
						`#define LEEP_CODE_1 ""`,
						`#define LEEP_CODE_2 ""`,
						"",
					}, "\n"),
				},
			},
			readFileResponses: []*readFileResponse{{
				// Copy file read
				expectedFile: filepath.Join(qw().QMKDir, "kb_km.bin"),
				contents:     "abcd",
			}},
			etc: &commandtest.ExecuteTestCase{
				Args: []string{
					"kb",
					"km",
					"--codes",
					"~~~~",
					"~",
					"--hash",
				},
				RunResponses: []*commandtest.FakeRun{
					{
						Stdout: []string{"abc"},
					},
					{
						Stdout: []string{"so"},
						Stderr: []string{"se"},
					},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					keyboardArg.Name(): "kb",
					keymapArg.Name():   "km",
					codesFlag.Name():   []string{"~~~~", "~"},
					hashFlag.Name():    true,
					hexFileFlag.Name(): "bin",
					"VERSION":          "abc",
				}},
				WantRunContents: []*commandtest.RunContents{
					{
						Name: "git",
						Args: []string{"rev-parse", "HEAD"},
						Dir:  qw().QMKDir,
					},
					{
						Name: "qmk",
						Args: []string{
							"compile",
							"--keyboard", "kb",
							"--keymap", "km",
						},
					},
				},
				WantStdout: "so\n",
				WantStderr: "se\n",
			},
		},
		{
			name: "succeeds with rot",
			q:    qwHash("abcd", "1234"),
			writeFileResponses: []*writeFileResponse{
				// Write codes to file
				{
					expectedFile: filepath.Join(qw().QMKDir, codeFile),
					expectedData: strings.Join([]string{
						"#pragma once",
						`#define LEEP_VERSION "abc123"`,
						//                    abcd (offsets, 1, 2, 3, 1)
						`#define LEEP_CODE_1 "bdfe"`,
						//                    1234 (offsets, 1, 1, 1, 1)
						`#define LEEP_CODE_2 "2345"`,
						"",
					}, "\n"),
				},
				// Copy write
				{
					expectedFile: filepath.Join(qw().OutputDir, "kb_km.bin"),
					expectedData: "abcd",
				},
				// Write empty strings to file
				{
					expectedFile: filepath.Join(qw().QMKDir, codeFile),
					expectedData: strings.Join([]string{
						"#pragma once",
						`#define LEEP_VERSION "auto-generated"`,
						`#define LEEP_CODE_1 ""`,
						`#define LEEP_CODE_2 ""`,
						"",
					}, "\n"),
				},
			},
			readFileResponses: []*readFileResponse{{
				// Copy file read
				expectedFile: filepath.Join(qw().QMKDir, "kb_km.bin"),
				contents:     "abcd",
			}},
			etc: &commandtest.ExecuteTestCase{
				Args: []string{
					"kb",
					"km",
					"--codes",
					fmt.Sprintf("%c%c%c", minRune+1, minRune+2, minRune+3),
					fmt.Sprintf("%c%c%c%c%c", minRune+1, minRune+1, minRune+1, minRune+1, minRune+1),
					"--hash",
				},
				RunResponses: []*commandtest.FakeRun{
					{
						Stdout: []string{"abc123"},
					},
					{
						Stdout: []string{"so"},
						Stderr: []string{"se"},
					},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					keyboardArg.Name(): "kb",
					keymapArg.Name():   "km",
					codesFlag.Name():   []string{`!"#`, "!!!!!"},
					hashFlag.Name():    true,
					hexFileFlag.Name(): "bin",
					"VERSION":          "abc123",
				}},
				WantRunContents: []*commandtest.RunContents{
					{
						Name: "git",
						Args: []string{"rev-parse", "HEAD"},
						Dir:  qw().QMKDir,
					},
					{
						Name: "qmk",
						Args: []string{
							"compile",
							"--keyboard", "kb",
							"--keymap", "km",
						},
					},
				},
				WantStdout: "so\n",
				WantStderr: "se\n",
			},
		},
		{
			name: "succeeds with rot and empty code",
			q:    qwHash("abcd", "1234"),
			writeFileResponses: []*writeFileResponse{
				// Write codes to file
				{
					expectedFile: filepath.Join(qw().QMKDir, codeFile),
					expectedData: strings.Join([]string{
						"#pragma once",
						`#define LEEP_VERSION "abc123"`,
						`#define LEEP_CODE_1 ""`,
						//                    1234 (offsets, 1, 1, 1, 1)
						`#define LEEP_CODE_2 "2345"`,
						"",
					}, "\n"),
				},
				// Copy write
				{
					expectedFile: filepath.Join(qw().OutputDir, "kb_km.bin"),
					expectedData: "abcd",
				},
				// Write empty strings to file
				{
					expectedFile: filepath.Join(qw().QMKDir, codeFile),
					expectedData: strings.Join([]string{
						"#pragma once",
						`#define LEEP_VERSION "auto-generated"`,
						`#define LEEP_CODE_1 ""`,
						`#define LEEP_CODE_2 ""`,
						"",
					}, "\n"),
				},
			},
			readFileResponses: []*readFileResponse{{
				// Copy file read
				expectedFile: filepath.Join(qw().QMKDir, "kb_km.bin"),
				contents:     "abcd",
			}},
			etc: &commandtest.ExecuteTestCase{
				Args: []string{
					"kb",
					"km",
					"--codes",
					"",
					fmt.Sprintf("%c%c%c%c%c", minRune+1, minRune+1, minRune+1, minRune+1, minRune+1),
					"--hash",
				},
				RunResponses: []*commandtest.FakeRun{
					{
						Stdout: []string{"abc123"},
					},
					{
						Stdout: []string{"so"},
						Stderr: []string{"se"},
					},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					keyboardArg.Name(): "kb",
					keymapArg.Name():   "km",
					codesFlag.Name():   []string{``, "!!!!!"},
					hashFlag.Name():    true,
					hexFileFlag.Name(): "bin",
					"VERSION":          "abc123",
				}},
				WantRunContents: []*commandtest.RunContents{
					{
						Name: "git",
						Args: []string{"rev-parse", "HEAD"},
						Dir:  qw().QMKDir,
					},
					{
						Name: "qmk",
						Args: []string{
							"compile",
							"--keyboard", "kb",
							"--keymap", "km",
						},
					},
				},
				WantStdout: "so\n",
				WantStderr: "se\n",
			},
		},
		{
			name: "succeeds with re-write error",
			q:    qw(),
			writeFileResponses: []*writeFileResponse{
				// Write codes to file
				{
					expectedFile: filepath.Join(qw().QMKDir, codeFile),
					expectedData: strings.Join([]string{
						"#pragma once",
						`#define LEEP_VERSION "abc123"`,
						`#define LEEP_CODE_1 "message 1"`,
						`#define LEEP_CODE_2 "message two"`,
						"",
					}, "\n"),
				},
				// Copy write
				{
					expectedFile: filepath.Join(qw().OutputDir, "kb_km.bin"),
					expectedData: "abcd",
				},
				// Write empty strings to file
				{
					expectedFile: filepath.Join(qw().QMKDir, codeFile),
					expectedData: strings.Join([]string{
						"#pragma once",
						`#define LEEP_VERSION "auto-generated"`,
						`#define LEEP_CODE_1 ""`,
						`#define LEEP_CODE_2 ""`,
						"",
					}, "\n"),
					err: fmt.Errorf("nooooo"),
				},
			},
			readFileResponses: []*readFileResponse{{
				// Copy file read
				expectedFile: filepath.Join(qw().QMKDir, "kb_km.bin"),
				contents:     "abcd",
			}},
			etc: &commandtest.ExecuteTestCase{
				Args: []string{
					"kb",
					"km",
					"--codes",
					"message 1",
					"message two",
				},
				RunResponses: []*commandtest.FakeRun{
					{
						Stdout: []string{"abc123def456"},
					},
					{
						Stdout: []string{"so"},
						Stderr: []string{"se"},
					},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					keyboardArg.Name(): "kb",
					keymapArg.Name():   "km",
					codesFlag.Name():   []string{"message 1", "message two"},
					hexFileFlag.Name(): "bin",
					"VERSION":          "abc123def456",
				}},
				WantRunContents: []*commandtest.RunContents{
					{
						Name: "git",
						Args: []string{"rev-parse", "HEAD"},
						Dir:  qw().QMKDir,
					},
					{
						Name: "qmk",
						Args: []string{
							"compile",
							"--keyboard", "kb",
							"--keymap", "km",
						},
					},
				},
				WantStdout: "so\n",
				WantStderr: strings.Join([]string{
					"se",
					"CRITICAL: failed to remove temporary codes: nooooo",
					"",
				}, "\n"),
			},
		},
		{
			name: "succeeds when no codes provided",
			q:    qw(),
			readFileResponses: []*readFileResponse{{
				// Copy file read
				expectedFile: filepath.Join(qw().QMKDir, "kb_km.bin"),
				contents:     "abcd",
			}},
			writeFileResponses: []*writeFileResponse{
				// Write codes to file
				{
					expectedFile: filepath.Join(qw().QMKDir, codeFile),
					expectedData: strings.Join([]string{
						"#pragma once",
						`#define LEEP_VERSION "abc123"`,
						`#define LEEP_CODE_1 ""`,
						`#define LEEP_CODE_2 ""`,
						"",
					}, "\n"),
				},
				// Copy write
				{
					expectedFile: filepath.Join(qw().OutputDir, "kb_km.bin"),
					expectedData: "abcd",
				},
				// Write empty strings to file
				{
					expectedFile: filepath.Join(qw().QMKDir, codeFile),
					expectedData: strings.Join([]string{
						"#pragma once",
						`#define LEEP_VERSION "auto-generated"`,
						`#define LEEP_CODE_1 ""`,
						`#define LEEP_CODE_2 ""`,
						"",
					}, "\n"),
				},
			},
			etc: &commandtest.ExecuteTestCase{
				Args: []string{
					"kb",
					"km",
				},
				RunResponses: []*commandtest.FakeRun{
					{
						Stdout: []string{"abc123def456"},
					},
					{
						Stdout: []string{"so"},
						Stderr: []string{"se"},
					},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					keyboardArg.Name(): "kb",
					keymapArg.Name():   "km",
					hexFileFlag.Name(): "bin",
					"VERSION":          "abc123def456",
				}},
				WantRunContents: []*commandtest.RunContents{
					{
						Name: "git",
						Args: []string{"rev-parse", "HEAD"},
						Dir:  qw().QMKDir,
					},
					{
						Name: "qmk",
						Args: []string{
							"compile",
							"--keyboard", "kb",
							"--keymap", "km",
						},
					},
				},
				WantStdout: "so\n",
				WantStderr: "se\n",
			},
		},
		// Config tests
		{
			name: "lists config",
			q:    qw(),
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"config", "list"},
				WantStdout: strings.Join([]string{
					fmt.Sprintf("QMK Directory:    %s", qw().QMKDir),
					fmt.Sprintf("Output Directory: %s", qw().OutputDir),
					"",
				}, "\n"),
			},
		},
		{
			name: "Writes config",
			q:    qw(),
			want: &qmkWrapper{
				QMKDir:    commandtest.FilepathAbs(t, filepath.Join("testdata", "qmk")),
				OutputDir: commandtest.FilepathAbs(t, filepath.Join("testdata", "out", "put")),
			},
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"config", "set", filepath.Join("testdata", "qmk"), filepath.Join("testdata", "out", "put")},
				WantData: &command.Data{Values: map[string]interface{}{
					qmkDirArg.Name():    commandtest.FilepathAbs(t, filepath.Join("testdata", "qmk")),
					outputDirArg.Name(): commandtest.FilepathAbs(t, filepath.Join("testdata", "out", "put")),
				}},
			},
		},
		// Shortcut tests (only need one test; assume all other logic works based on tests in command package)
		{
			name: "Adds shortcut",
			q:    qw(),
			want: &qmkWrapper{
				QMKDir:    qw().QMKDir,
				OutputDir: qw().OutputDir,
				Shortcuts: map[string]map[string][]string{
					shortcutName: {
						"p": []string{"kb/subkb", "km", "--codes", "shortcut-message-1", "msg2"},
					},
				},
			},
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"shortcuts", "add", "p", "kb/subkb", "km", "--codes", "shortcut-message-1", "msg2"},
				RunResponses: []*commandtest.FakeRun{
					{
						Stdout: []string{"abc123def456"},
					},
				},
				WantRunContents: []*commandtest.RunContents{
					{
						Name: "git",
						Args: []string{"rev-parse", "HEAD"},
						Dir:  qw().QMKDir,
					},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					keyboardArg.Name():           "kb/subkb",
					keymapArg.Name():             "km",
					codesFlag.Name():             []string{"shortcut-message-1", "msg2"},
					commander.ShortcutArg.Name(): "p",
					hexFileFlag.Name():           "bin",
					"VERSION":                    "abc123def456",
				}},
			},
		},
		/* Useful for commenting out tests. */
	} {
		t.Run(test.name, func(t *testing.T) {
			test.etc.Node = test.q.Node()

			commandtest.StubValue(t, &osReadFile, func(s string) ([]byte, error) {
				if test.readFileResponses == nil {
					t.Fatalf("Ran out of readFileResponses")
				}
				res := test.readFileResponses[0]

				if diff := cmp.Diff(res.expectedFile, s); diff != "" {
					t.Fatalf("osReadFile() called with wrong file (-want, +got):\n%s", diff)
				}

				test.readFileResponses = test.readFileResponses[1:]
				return []byte(res.contents), res.err
			})

			commandtest.StubValue(t, &osWriteFile, func(s string, data []byte, _ os.FileMode) error {
				if test.writeFileResponses == nil {
					t.Fatalf("Ran out of writeFileResponses")
				}
				res := test.writeFileResponses[0]

				if diff := cmp.Diff(res.expectedFile, s); diff != "" {
					t.Fatalf("osWriteFile() called with wrong file (-want, +got):\n%s", diff)
				}
				if diff := cmp.Diff(res.expectedData, string(data)); diff != "" {
					t.Fatalf("osWriteFile() called with wrong contents (-want, +got):\n%s", diff)
				}

				test.writeFileResponses = test.writeFileResponses[1:]
				return res.err
			})

			commandertest.ExecuteTest(t, test.etc)
			commandertest.ChangeTest(t, test.want, test.q, cmpopts.IgnoreUnexported(qmkWrapper{}), cmpopts.EquateEmpty())
		})
	}
}

func TestMetadata(t *testing.T) {
	qw := &qmkWrapper{}
	if diff := cmp.Diff("q", qw.Name()); diff != "" {
		t.Errorf("qmkWrapper.Name() returned wrong value (-want, +got):\n%s", diff)
	}

	if diff := cmp.Diff([]string(nil), qw.Setup()); diff != "" {
		t.Errorf("qmkWrapper.Setup() returned wrong value (-want, +got):\n%s", diff)
	}

	// CLI() doesn't error
	CLI("abc", "")
}

func TestRot(t *testing.T) {
	for _, test := range []struct {
		name     string
		positive string
		negative string
		key      string
	}{
		{
			name:     "simple rot",
			positive: "12345678",
			negative: "rv.Hvz2L",
			key:      "ady4",
		},
		{
			name:     "rot with length mismat h",
			positive: "12345678",
			negative: "rv.How{3",
			key:      "ady4Z",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if posRot := rot(test.positive, test.key, true); posRot != test.negative {
				t.Errorf("rot(%s, %s, true) returned %s; wanted %s", test.positive, test.key, posRot, test.negative)
			}
			if negRot := rot(test.negative, test.key, false); negRot != test.positive {
				t.Errorf("rot(%s, %s, false) returned %s; wanted %s", test.negative, test.key, negRot, test.positive)
			}
		})
	}
}
