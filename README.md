# ku-builder

Cmake / Makefile build util using [j9](https://github.com/mgenware/j9) for C/C++ projects.

Supported cross-compile targets:

- macOS (ARM64, x86_64)
- iOS (ARM64, x86_64)
- iOS Simulator (ARM64, x86_64)
- Android (ARM64, x86_64)

Minimum SDK versions:

- macOS 11+
- iOS 14+
- Android SDK API level 26+

Supported host platform:

- Latest stable macOS version.

## ku-builder Utils CLI (kbu)

### Installation

```bash
go install github.com/mgenware/ku-builder/cmd/kbu@latest
```

### Usage

```
kbu <action> <a dylib or so file>
```

Supported actions:

- `deps` Show dependencies.
