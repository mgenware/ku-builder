package ku

import (
	"flag"
	"fmt"
	"os"
	"slices"

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
	SignArg     string
	PlatformArg PlatformEnum

	Options *CLIOptions
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
	AllowedTargets  []string
	DefaultPlatform PlatformEnum
	DefaultTarget   string
	DefaultSDK      SDKEnum
	DefaultArch     ArchEnum
	DefaultAction   CLIAction
	CreateDistDir   bool

	BeforeParseFn func()
	AfterParseFn  func(cliArgs *CLIArgs)
}

func ParseCLIArgs(opt *CLIOptions) *CLIArgs {
	if opt == nil {
		fmt.Printf("CLIOptions is nil\n")
		os.Exit(1)
	}

	var allowedTargets []string
	if len(opt.AllowedTargets) > 0 {
		allowedTargets = opt.AllowedTargets
	} else if opt.DefaultTarget != "" {
		allowedTargets = []string{opt.DefaultTarget}
	} else {
		fmt.Printf("AllowedTargets is empty and DefaultTarget is not set\n")
		os.Exit(1)
	}

	platformPtr := flag.String("platform", string(opt.DefaultPlatform), "Platform. Supported platforms: macos, ios, android, darwin(macos + ios).")
	target := flag.String("target", opt.DefaultTarget, "Build target. "+"Allowed targets: "+fmt.Sprintf("%v", allowedTargets))
	sdkPtr := flag.String("sdk", string(opt.DefaultSDK), "SDK. If not specified, all supported SDKs for the platform will be used.")
	archPtr := flag.String("arch", string(opt.DefaultArch), "Arch. If not specified, all supported SDK archs for the platform will be used.")
	actionPtr := flag.String("action", string(opt.DefaultAction), "Action. Supported actions: configure, clean, build.")
	ndkPtr := flag.String("ndk", "", "NDK name.")
	debugPtr := flag.Bool("debug", false, "Debug build.")
	cleanPtr := flag.Bool("clean", false, "Run a clean build.")
	signPtr := flag.String("sign", "", "Sign the output with the specified identity.")
	if opt.BeforeParseFn != nil {
		opt.BeforeParseFn()
	}

	flag.Parse()

	if *target == "" {
		fmt.Printf("Target is required\n")
		os.Exit(1)
	}
	if !slices.Contains(allowedTargets, *target) {
		fmt.Printf("Target %v is not allowed. Allowed targets: %v\n", *target, allowedTargets)
		os.Exit(1)
	}

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

	// There must be at least one sdk.
	// Note: `sdks` could be set by `-platform` or `-sdk`.
	if len(sdks) == 0 {
		fmt.Printf("No SDKs found. Please specify SDKs via -platform or -sdk.\n")
		os.Exit(1)
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
	if sdks[0] == SDKAndroid {
		if *ndkPtr == "" {
			fmt.Printf("NDK is not specified\n")
			os.Exit(1)
		}
	}

	res := &CLIArgs{
		SDKs:        sdks,
		Arch:        ArchEnum(*archPtr),
		Target:      *target,
		Action:      CLIAction(*actionPtr),
		DebugBuild:  *debugPtr,
		CleanBuild:  *cleanPtr,
		NDK:         *ndkPtr,
		SignArg:     *signPtr,
		PlatformArg: PlatformEnum(*platformPtr),
		Options:     opt,
	}

	if opt.AfterParseFn != nil {
		opt.AfterParseFn(res)
	}

	return res
}

func CreateDefaultTunnel() *j9.Tunnel {
	return j9.NewTunnel(j9.NewLocalNode(), j9.NewConsoleLogger())
}
