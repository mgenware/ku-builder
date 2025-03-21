package ku

import (
	"flag"
	"fmt"
	"os"

	"github.com/mgenware/j9/v3"
)

type CLIArgs struct {
	SDKs        []SDKEnum
	Arch        ArchEnum
	Target      string
	Action      CLIAction
	DebugBuild  bool
	NDK         string
	CleanBuild  bool
	Dylib       bool
	SignArg     string
	PlatformArg PlatformEnum
}

type CLIAction string

const (
	// Run ./configure
	CLIActionConfigure CLIAction = "configure"
	// Run make clean
	CLIActionClean CLIAction = "clean"
	// Run make && make install
	CLIActionBuild CLIAction = "build"
	// Run make
	CLIActionMake CLIAction = "make"
)

var SupportedCLIActions = map[CLIAction]bool{
	CLIActionConfigure: true,
	CLIActionClean:     true,
	CLIActionBuild:     true,
}

type CLIOptions struct {
	DefaultPlatform PlatformEnum
	DefaultTarget   string
	DefaultSDK      SDKEnum
	DefaultArch     ArchEnum
	DefaultAction   CLIAction
	DefaultDylib    bool
}

func ParseCLIArgs(opt *CLIOptions) *CLIArgs {
	if opt == nil {
		opt = &CLIOptions{}
	}

	platformPtr := flag.String("platform", string(opt.DefaultPlatform), "Platform. Supported platforms: macos, ios, android, darwin(macos + ios).")
	target := flag.String("target", opt.DefaultTarget, "Build target.")
	sdkPtr := flag.String("sdk", string(opt.DefaultSDK), "SDK. If not specified, all supported SDKs for the platform will be used.")
	archPtr := flag.String("arch", string(opt.DefaultArch), "Arch. If not specified, all supported SDK archs for the platform will be used.")
	actionPtr := flag.String("action", string(opt.DefaultAction), "Action. Supported actions: configure, clean, build.")
	dylibPtr := flag.Bool("dylib", opt.DefaultDylib, "Build as dylib.")
	ndkPtr := flag.String("ndk", "", "NDK name.")
	debugPtr := flag.Bool("debug", false, "Debug build.")
	cleanPtr := flag.Bool("clean", false, "Run a clean build.")
	signPtr := flag.String("sign", "", "Sign the output with the specified identity.")

	flag.Parse()

	var sdks []SDKEnum
	// Validate platform if specified.
	if *platformPtr != "" {
		platform := PlatformEnum(*platformPtr)
		if !SupportedPlatforms[platform] {
			fmt.Printf("Unsupported platform: %v\n", string(platform))
			os.Exit(1)
		}
		sdks = PlatformSDKs[platform]
		if sdks == nil {
			fmt.Printf("No supported SDKs for platform: %v\n", string(platform))
			os.Exit(1)
		}
	}
	// Validate sdk.
	if *sdkPtr != "" {
		if !SupportedSDKs[SDKEnum(*sdkPtr)] {
			fmt.Printf("Unsupported SDK: %v\n", *sdkPtr)
			os.Exit(1)
		}
		if sdks != nil {
			fmt.Printf("Both -sdk and -platform are specified\n")
		}
		sdks = []SDKEnum{SDKEnum(*sdkPtr)}
	}
	// Validate arch.
	if *archPtr != "" {
		if !SupportedArchs[ArchEnum(*archPtr)] {
			fmt.Printf("Unsupported arch: %v\n", *archPtr)
			os.Exit(1)
		}
	}
	// Validate action.
	if *actionPtr == "" {
		if !SupportedCLIActions[CLIActionBuild] {
			fmt.Printf("Unsupported action: %v\n", *actionPtr)
			os.Exit(1)
		}
	}
	// Validate Android settings.
	if len(sdks) > 0 && sdks[0] == SDKAndroid {
		if *ndkPtr == "" {
			fmt.Printf("NDK is not specified\n")
			os.Exit(1)
		}
	}

	return &CLIArgs{
		SDKs:        sdks,
		Arch:        ArchEnum(*archPtr),
		Target:      *target,
		Action:      CLIAction(*actionPtr),
		DebugBuild:  *debugPtr,
		CleanBuild:  *cleanPtr,
		NDK:         *ndkPtr,
		SignArg:     *signPtr,
		Dylib:       *dylibPtr,
		PlatformArg: PlatformEnum(*platformPtr),
	}
}

func CreateDefaultTunnel() *j9.Tunnel {
	return j9.NewTunnel(j9.NewLocalNode(), j9.NewConsoleLogger())
}
