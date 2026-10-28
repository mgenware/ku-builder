package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestWriteUsage(t *testing.T) {
	var out bytes.Buffer
	writeUsage(&out)

	for _, want := range []string{
		"Usage: kuu [options] <action> [input]",
		"Put options before the action",
		"dep <input>",
		"symbol <input>",
		"deploy",
		"-platform, -p",
		"-target, -t",
		"-ndk",
		"-debug, -d",
		"-verbose",
		"-help, -h",
		"ANDROID_HOME",
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("usage is missing %q", want)
		}
	}
}
