package qmkwrapper

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/leep-frog/command/command"
	"github.com/leep-frog/command/commander"
	"github.com/leep-frog/command/sourcerer"
)

const (
	// TODO: Use this to replace old qmk CLI.
	QMKEnvArg = "LEEP_QMK"
)

var (
	// Var so can stub out in tests
	timeNow = time.Now
)

var (
	shortcutName = "compile-shortcut"
	codeFile     = filepath.Join("users", "leep-frog", "v2", "leep_codes_v2.h")
	slashRegbex  = regexp.MustCompile(`[\\/]`)
	// methods that are stubbed in tests
	osReadFile  = os.ReadFile
	osWriteFile = os.WriteFile

	// TODO: Actualy use these binding things to replace the old qmk CLI.
	basicKeyboardBindings = []string{
		`bind '"\C-h":backward-delete-char'`,
	}
	qmkKeyboardBindings = []string{
		`bind '"\C-h":backward-kill-word'`,
	}
)

func CLI(code1, code2 string) sourcerer.CLI {
	return &qmkWrapper{
		hash:  code1,
		hash2: code2,
	}
}

func Aliasers() sourcerer.Option {
	return sourcerer.Aliasers(map[string][]string{
		"qm":  {"q", "m"},
		"qk":  {"q", "k"},
		"qp":  {"q", "p"},
		"qgr": {"q", "gr"},
	})
}

func codeFileContents(version, code1, code2 string) []byte {
	return []byte(strings.Join([]string{
		"#pragma once",
		fmt.Sprintf("#define LEEP_VERSION %q", version),
		fmt.Sprintf("#define LEEP_CODE_1 %q", code1),
		fmt.Sprintf("#define LEEP_CODE_2 %q", code2),
		"",
	}, "\n"))
}

const (
	minRune           = 32
	maxRune           = 126
	normalizedMaxRune = maxRune - minRune
)

func rot(hash, key string, pos bool) string {
	if len(key) == 0 {
		return ""
	}
	var r []string
	// offset start at min char (rune 0?)
	for i, c := range hash {
		k := rune(key[i%len(key)])
		// normalized c and k
		nc := c - minRune
		nk := k - minRune

		var nv rune
		if pos {
			nv = (nc + nk) % (normalizedMaxRune)
		} else {
			nv = (nc + normalizedMaxRune - nk) % (normalizedMaxRune)
		}

		// regular v
		v := nv + minRune
		r = append(r, fmt.Sprintf("%c", v))
	}
	return strings.Join(r, "")
}

type qmkWrapper struct {
	QMKDir    string
	OutputDir string
	Shortcuts map[string]map[string][]string

	hash    string
	hash2   string
	changed bool
}

func (qw *qmkWrapper) Name() string {
	return "q"
}

func (qw *qmkWrapper) Changed() bool {
	return qw.changed
}

func (qw *qmkWrapper) Setup() []string {
	return nil
}

var (
	// Compile args
	keyboardArg = commander.Arg[string]("KEYBOARD", "Keyboard")
	keymapArg   = commander.Arg[string]("KEYMAP", "Keymap")
	hexFileFlag = commander.BoolValuesFlag("hex-file", 'x', "If the suffix is a hex file", "hex", "bin")
	hashFlag    = commander.BoolFlag("hash", 'h', "Whether code1 and code2 should be hashed")
	codesFlag   = commander.ListFlag[string]("codes", 'c', "Codes for fixed code keys", 2, 0)

	// Config args
	qmkDirArg = commander.FileArgument("QMK_DIR", "Root directory of QMK", commander.IsDir(), &commander.FileCompleter[string]{
		IgnoreFiles: true,
	})
	outputDirArg = commander.FileArgument("OUTPUT_DIR", "Output directory for qmk compilation artifacts", commander.IsDir(), &commander.FileCompleter[string]{
		IgnoreFiles: true,
	})
)

func (qw *qmkWrapper) MarkChanged() { qw.changed = true }

func (qw *qmkWrapper) ShortcutMap() map[string]map[string][]string {
	if qw.Shortcuts == nil {
		qw.Shortcuts = map[string]map[string][]string{}
	}
	return qw.Shortcuts
}

func (qw *qmkWrapper) Node() command.Node {
	verifyConfig := commander.SuperSimpleProcessor(func(i *command.Input, d *command.Data) error {
		if qw.QMKDir == "" || qw.OutputDir == "" {
			return fmt.Errorf("Directory values have not been set (`q config set`)")
		}
		return nil
	})
	versionCommand := &commander.ShellCommand[string]{
		ArgName:     "VERSION",
		CommandName: "git",
		Args:        []string{"rev-parse", "HEAD"},
		Dir:         qw.QMKDir,
	}
	return &commander.BranchNode{
		Branches: map[string]command.Node{
			"test": commander.SerialNodes(
				commander.SimpleExecutableProcessor("make test:leep_frog"),
			),
			"config": &commander.BranchNode{
				Branches: map[string]command.Node{
					"list": commander.SerialNodes(
						&commander.ExecutorProcessor{func(o command.Output, d *command.Data) error {
							o.Stdoutf("QMK Directory:    %s\n", qw.QMKDir)
							o.Stdoutf("Output Directory: %s\n", qw.OutputDir)
							return nil
						}},
					),
					"set": commander.SerialNodes(
						qmkDirArg,
						outputDirArg,
						&commander.ExecutorProcessor{func(o command.Output, d *command.Data) error {
							qw.QMKDir = qmkDirArg.Get(d)
							qw.OutputDir = outputDirArg.Get(d)
							qw.changed = true
							return nil
						}},
					),
				},
			},
		},
		Default: commander.ShortcutNode(shortcutName, qw, commander.SerialNodes(
			verifyConfig,
			commander.FlagProcessor(
				hexFileFlag,
				hashFlag,
				codesFlag,
			),
			keyboardArg,
			keymapArg,
			versionCommand,
			&commander.ExecutorProcessor{func(o command.Output, d *command.Data) error {

				kb := keyboardArg.Get(d)
				km := keymapArg.Get(d)

				version := versionCommand.Get(d)
				if len(version) > 6 {
					version = version[:6]
				}

				var code1, code2 string
				if codesFlag.Provided(d) {
					codes := codesFlag.Get(d)
					code1, code2 = codes[0], codes[1]
				}

				if hashFlag.Get(d) {
					code1 = rot(qw.hash, code1, true)
					code2 = rot(qw.hash2, code2, true)
				}

				timedVersion := timeNow().Format("2006-01-02 15:04:05 ") + version
				if err := osWriteFile(filepath.Join(qw.QMKDir, codeFile), []byte(codeFileContents(timedVersion, code1, code2)), 0644); err != nil {
					return o.Annotate(err, "failed to write code file")
				}

				defer func() {
					if err := osWriteFile(filepath.Join(qw.QMKDir, codeFile), []byte(codeFileContents("auto-generated", "", "")), 0644); err != nil {
						o.Annotatef(err, "CRITICAL: failed to remove temporary codes")
					}
				}()

				// Run the qmk command
				bc := &commander.ShellCommand[string]{
					CommandName: "qmk",
					Args: []string{
						"compile",
						"--keyboard", kb,
						"--keymap", km,
					},
					ForwardStdout: true,
				}
				if _, err := bc.Run(o, d); err != nil {
					return o.Annotate(err, "failed to run qmk compile")
				}

				// Copy the output file
				bf := fmt.Sprintf("%s_%s.%s", slashRegbex.ReplaceAllString(kb, "_"), slashRegbex.ReplaceAllString(km, "_"), hexFileFlag.Get(d))
				if err := copyFile(filepath.Join(qw.QMKDir, bf), filepath.Join(qw.OutputDir, bf)); err != nil {
					return o.Annotate(err, "failed to copy qmk files")
				}

				return nil
			}},
		)),
	}
}

func copyFile(from, to string) error {
	data, err := osReadFile(from)
	if err != nil {
		return fmt.Errorf("failed to read input file: %v", err)
	}
	if err := osWriteFile(to, data, 0644); err != nil {
		return fmt.Errorf("failed to write to output file: %v", err)
	}
	return nil
}
