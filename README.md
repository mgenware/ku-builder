# ku-builder

CMake / Makefile / Meson build util.

Supported cross-compiling targets:

- macOS (ARM64, x86_64)
- iOS (ARM64)
- iOS Simulator (ARM64)
- Android NDK (ARM64, x86_64)

Minimum cross-compiling SDK versions:

- macOS 11+
- iOS 14+
- Android SDK API level 26+

Supported host OS:

- Latest stable macOS.

## ku-builder Utils CLI (kbu)

### Installation

```bash
go install github.com/mgenware/ku-builder/kbu@latest
```

### Usage

```
Usage: kbu <action> [options] <input>

Actions:
  deps       List dependencies of the input file
  symbols    List exported symbols of the input file

Options:
  -ndk       Specify NDK version
  -os        Specify the operating system type: 'd' for Darwin, 'a' for Android
```
