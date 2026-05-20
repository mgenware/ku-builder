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

## ku-builder Utils CLI (kuu)

### Installation

```bash
go install github.com/mgenware/ku-builder/kuu@latest
```

### Usage

```
Usage: kuu [options] <action> <input>

Actions:
  dep       List dependencies of the input file
  symbol    List exported symbols of the input file
  deploy    Run deployment for the specified target and platform. Input is ignored.

Options:
  -platform  Platform. Supported platforms: macos(m), ios(i), android(a), darwin(d).
  -p         -platform shorthand.
  -target    Build target.
  -t         -target shorthand.
  -ndk       NDK version.
  -debug     Debug build.
  -help      Show usage information.
```
