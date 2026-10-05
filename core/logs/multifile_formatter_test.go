// Copyright 2026 beego Author. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package logs

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type multiFileFormatter struct{}

func (multiFileFormatter) Format(msg *LogMsg) string {
	return "custom:" + msg.Msg + "\n"
}

func TestMultiFileGlobalFormatter(t *testing.T) {
	const formatter = "multifile-regression"
	RegisterFormatter(formatter, multiFileFormatter{})
	oldLogger := beeLogger
	beeLogger = NewLogger()
	defer func() {
		Reset()
		beeLogger = oldLogger
		delete(formatterMap, formatter)
	}()
	for _, mode := range []string{"default", "adapter", "global"} {
		t.Run(mode, func(t *testing.T) {
			Reset()
			global := ""
			if mode == "global" {
				global = formatter
			}
			if err := SetGlobalFormatter(global); err != nil {
				t.Fatal(err)
			}
			levels := []string{"emergency", "alert", "critical", "error", "warning", "notice", "info", "debug"}
			filename := filepath.Join(t.TempDir(), "test.log")
			config := map[string]interface{}{"filename": filename, "separate": levels, "rotate": false}
			if mode == "adapter" {
				config["formatter"] = formatter
			}
			data, err := json.Marshal(config)
			if err != nil {
				t.Fatal(err)
			}
			if err := SetLogger(AdapterMultiFile, string(data)); err != nil {
				t.Fatal(err)
			}
			write := []func(interface{}, ...interface{}){Emergency, Alert, Critical, Error, Warning, Notice, Info, Debug}
			for i, log := range write {
				log("message-" + levels[i])
			}
			Reset() // Close the real files before reading them.
			full, err := os.ReadFile(filename)
			if err != nil {
				t.Fatal(err)
			}
			fullLines := strings.Split(strings.TrimSuffix(string(full), "\n"), "\n")
			if len(fullLines) != len(levels) {
				t.Fatalf("full log has %d lines, want %d", len(fullLines), len(levels))
			}
			for i, level := range levels {
				separate, err := os.ReadFile(strings.TrimSuffix(filename, ".log") + "." + level + ".log")
				if err != nil {
					t.Fatal(err)
				}
				line := fullLines[i]
				if !strings.Contains(line, "message-"+level) {
					t.Errorf("full log line %q missing message for %s", line, level)
				}
				if mode != "default" && !strings.HasPrefix(line, "custom:") {
					t.Errorf("full log did not use custom formatter: %q", line)
				}
				if mode == "default" && strings.HasPrefix(line, "custom:") {
					t.Errorf("default log unexpectedly used custom formatter: %q", line)
				}
				if string(separate) != line+"\n" {
					t.Errorf("%s log = %q, want %q", level, separate, line+"\n")
				}
			}
		})
	}
}
