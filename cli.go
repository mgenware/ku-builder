package ku

import (
	"flag"
	"fmt"
	"os"
	"slices"

	"github.com/mgenware/j9/v3"
)

type CLIArgs struct {
	SDKs            []SDKEnum
	Arch            ArchEnum
	Target          string
	Action          CLIAction
	DebugBuild      bool
	NDK             string
	CleanBuild      bool
	SignArg         string
	PlatformArg     PlatformEnum
	HardenedRuntime bool
	LibType         LibType
	NoPull          bool

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
	CLIActionMake:      true,
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

	var platformInput string
	var resolvedPlatform PlatformEnum
	flag.StringVar(&platformInput, "platform", string(opt.DefaultPlatform), "Select SDKs: macos (m), ios (i: device and simulator), android (a), or darwin (d: macOS and iOS).")
	flag.StringVar(&platformInput, "p", string(opt.DefaultPlatform), "-platform shorthand.")

	var target string
	flag.StringVar(&target, "target", opt.DefaultTarget, "Build target; required unless a default is configured. Allowed targets: "+fmt.Sprintf("%v", allowedTargets))
	flag.StringVar(&target, "t", opt.DefaultTarget, "-target shorthand.")

	sdkPtr := flag.String("sdk", string(opt.DefaultSDK), "Select one SDK: macosx, iphoneos, iphonesimulator, or android; overrides -platform.")
	archPtr := flag.String("arch", string(opt.DefaultArch), "Select arm64 or x86_64; otherwise use all supported architectures for each SDK.")
	actionPtr := flag.String("action", string(opt.DefaultAction), "Action for the calling program to handle: configure, clean, build, or make.")
	ndkPtr := flag.String("ndk", "", "Android NDK version or absolute path; required for Android builds.")

	var debug bool
	flag.BoolVar(&debug, "debug", false, "Use debug instead of release build directories and flags.")
	flag.BoolVar(&debug, "d", false, "-debug shorthand.")

	cleanPtr := flag.Bool("clean", false, "Clean temporary build directories before configuring.")
	dylibPtr := flag.Bool("dylib", false, "Request a dynamic/shared library instead of a static library (if supported by the calling program).")
	signPtr := flag.String("sign", "", "Signing identity for programs that sign their output.")
	hardenedRuntimePtr := flag.Bool("hardened", false, "Enable hardened runtime when signing macOS frameworks.")
	noPullPtr := flag.Bool("no-pull", false, "Skip git pull for existing source checkouts.")
	if opt.BeforeParseFn != nil {
		opt.BeforeParseFn()
	}

	flag.Parse()

	if target == "" {
		fmt.Printf("Target is required\n")
		os.Exit(1)
	}
	if !slices.Contains(allowedTargets, target) {
		fmt.Printf("Target %v is not allowed. Allowed targets: %v\n", target, allowedTargets)
		os.Exit(1)
	}

	if platformInput == "" && *ndkPtr != "" {
		// Make platform default to android if NDK is specified.
		platformInput = string(PlatformAndroid)
	}

	var sdks []SDKEnum
	// Validate platform if specified.
	if platformInput != "" {
		resolvedPlatform = ParsePlatformString(platformInput, false)
		sdks = PlatformSDKs[resolvedPlatform]
		if sdks == nil {
			fmt.Printf("No supported SDKs for platform: %v\n", string(resolvedPlatform))
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
	if *actionPtr != "" {
		if !SupportedCLIActions[CLIAction(*actionPtr)] {
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

	libType := LibTypeStatic
	if *dylibPtr {
		libType = LibTypeDynamic
	}

	res := &CLIArgs{
		SDKs:            sdks,
		Arch:            ArchEnum(*archPtr),
		Target:          target,
		Action:          CLIAction(*actionPtr),
		DebugBuild:      debug,
		CleanBuild:      *cleanPtr,
		NDK:             *ndkPtr,
		SignArg:         *signPtr,
		PlatformArg:     resolvedPlatform,
		LibType:         libType,
		Options:         opt,
		NoPull:          *noPullPtr,
		HardenedRuntime: *hardenedRuntimePtr,
	}

	if opt.AfterParseFn != nil {
		opt.AfterParseFn(res)
	}

	return res
}

func CreateDefaultTunnel() *j9.Tunnel {
	return j9.NewTunnel(j9.NewLocalNode(), j9.NewConsoleLogger())
}

func ParsePlatformString(s string, required bool) PlatformEnum {
	// Check shorthand first.
	switch s {
	case "m":
		return PlatformMacos
	case "i":
		return PlatformIos
	case "a":
		return PlatformAndroid
	case "d":
		return PlatformDarwin
	}

	// Check full string.
	switch s {
	case string(PlatformMacos):
		return PlatformMacos
	case string(PlatformIos):
		return PlatformIos
	case string(PlatformAndroid):
		return PlatformAndroid
	case string(PlatformDarwin):
		return PlatformDarwin
	default:
		if required {
			fmt.Printf("Unsupported platform: %v\n", s)
			os.Exit(1)
		}
	}
	return ""
}
