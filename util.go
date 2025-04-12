package ku

import (
	"path/filepath"
	"slices"

	"github.com/mgenware/j9/v3"
	"github.com/mgenware/ku-builder/io2"
)

func CopyJNILibs(cliArgs *CLIArgs, tunnel *j9.Tunnel, libFileNames []string, headerFileNames []string, relPath string) {
	if !slices.Contains(cliArgs.SDKs, SDKAndroid) {
		return
	}
	buildDir := GetBuildDir(cliArgs.DebugBuild)
	sdkDir := GetSDKDir(buildDir, SDKAndroid)

	jniBuildDir := filepath.Join(sdkDir, "jni")
	libsDir := filepath.Join(jniBuildDir, "jniLibs")
	includeDir := filepath.Join(jniBuildDir, "include")

	io2.Mkdirp(libsDir)
	io2.Mkdirp(includeDir)

	// Copy the JNI libs to the jniLibs directory.
	jniArchList := []ArchEnum{
		ArchArm64,
		ArchX86_64,
	}

	for _, arch := range jniArchList {
		for _, libFileName := range libFileNames {
			srcLibFile := filepath.Join(sdkDir, string(arch), relPath, "lib", libFileName)

			var jniArch string
			if arch == ArchArm64 {
				jniArch = "arm64-v8a"
			} else {
				jniArch = "x86_64"
			}
			jniArchDir := filepath.Join(libsDir, jniArch)
			io2.Mkdirp(jniArchDir)

			// Copy the lib file to the jniLibs directory.
			tunnel.Spawn(&j9.SpawnOpt{
				Name: "cp",
				Args: []string{srcLibFile, jniArchDir + "/"}},
			)
		}
	}

	headerSrcDir := filepath.Join(sdkDir, string(ArchArm64), relPath, "include")
	for _, headerFileName := range headerFileNames {
		srcHeaderFile := filepath.Join(headerSrcDir, headerFileName)

		// Copy the header file to the include directory.
		tunnel.Spawn(&j9.SpawnOpt{
			Name: "cp",
			Args: []string{"-R", srcHeaderFile, includeDir + "/"}},
		)
	}
}
