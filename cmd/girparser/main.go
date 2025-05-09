// Command-line tool for parsing libvips GIR files and generating C wrapper functions
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cshum/vipsgen/girparser"
)

func main() {
	// Set up logging
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Command line flags
	inputFlag := flag.String("input", "", "Input GIR file path (required)")
	outputHeaderFlag := flag.String("header", "vips_wrapper.h", "Output C header file path")
	outputImplFlag := flag.String("impl", "vips_wrapper.c", "Output C implementation file path")
	filterFlag := flag.String("filter", "", "Regex pattern to filter function names")
	verboseFlag := flag.Bool("verbose", false, "Print verbose information during processing")
	outputOnlyFlag := flag.Bool("output-only", false, "Output to stdout instead of files")
	debugFlag := flag.Bool("debug", false, "Print detailed debug information")
	dumpCIdentifiersFlag := flag.Bool("dump-cidentifiers", false, "Dump all C identifiers found in the GIR file")
	ignoreIntrospectionFlag := flag.Bool("ignore-introspection", false, "Process functions even if they're marked as non-introspectable")

	flag.Parse()

	if *debugFlag {
		log.Println("Debug mode enabled")
	} else {
		// Disable logging if not in debug mode
		log.SetOutput(ioutil.Discard)
	}

	// Validate input
	if *inputFlag == "" {
		fmt.Println("Error: Input GIR file is required")
		flag.Usage()
		os.Exit(1)
	}

	// Try to find the GIR file if not a full path
	inputPath := *inputFlag
	if !filepath.IsAbs(inputPath) {
		// First try current directory
		if _, err := os.Stat(inputPath); os.IsNotExist(err) {
			// Then try standard locations
			path, err := girparser.FindGIRFile(inputPath)
			if err != nil {
				// If still not found, try reading from stdin if input is '-'
				if inputPath == "-" {
					if *verboseFlag {
						fmt.Println("Reading GIR data from stdin")
					}

					data, err := ioutil.ReadAll(os.Stdin)
					if err != nil {
						fmt.Printf("Error reading from stdin: %v\n", err)
						os.Exit(1)
					}

					gir, debugInfo, err := girparser.ParseGIR(strings.NewReader(string(data)))
					if err != nil {
						fmt.Printf("Error parsing GIR data: %v\n", err)
						os.Exit(1)
					}

					if *verboseFlag || *debugFlag {
						printDebugInfo(debugInfo)
					}

					processGIR(gir, *filterFlag, *outputHeaderFlag, *outputImplFlag, *verboseFlag, *outputOnlyFlag, *dumpCIdentifiersFlag, *ignoreIntrospectionFlag)
					return
				}

				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
			inputPath = path
		}
	}

	if *verboseFlag {
		fmt.Printf("Parsing GIR file: %s\n", inputPath)
	}

	// Parse GIR file
	gir, debugInfo, err := girparser.ParseGIRFile(inputPath)
	if err != nil {
		fmt.Printf("Error parsing GIR file: %v\n", err)
		os.Exit(1)
	}

	if *verboseFlag || *debugFlag {
		printDebugInfo(debugInfo)
	}

	processGIR(gir, *filterFlag, *outputHeaderFlag, *outputImplFlag, *verboseFlag, *outputOnlyFlag, *dumpCIdentifiersFlag, *ignoreIntrospectionFlag)
}

func printDebugInfo(debugInfo *girparser.DebugInfo) {
	fmt.Println("\n=== Debug Information ===")
	fmt.Printf("Top-level functions: %d\n", debugInfo.FunctionsFound)
	fmt.Printf("Class methods and functions: %d\n", debugInfo.ClassMethodsFound)
	fmt.Printf("Interface methods and functions: %d\n", debugInfo.InterfaceMethodsFound)
	fmt.Printf("Record methods and functions: %d\n", debugInfo.RecordMethodsFound)
	fmt.Printf("Total methods and functions: %d\n",
		debugInfo.FunctionsFound+debugInfo.ClassMethodsFound+
			debugInfo.InterfaceMethodsFound+debugInfo.RecordMethodsFound)
	fmt.Printf("Introspectable functions: %d\n", debugInfo.IntrospectableFunctions)
	fmt.Printf("Non-introspectable functions: %d\n", debugInfo.NonIntrospectableFunctions)
	fmt.Printf("Functions without C identifier: %d\n", debugInfo.FunctionWithoutCIdentifier)
	fmt.Printf("Functions processed: %d\n", debugInfo.ProcessedFunctions)
	if debugInfo.NonIntrospectableIncluded > 0 {
		fmt.Printf("Non-introspectable functions included: %d\n", debugInfo.NonIntrospectableIncluded)
	}
	if debugInfo.MissingCIdentifierIncluded > 0 {
		fmt.Printf("Functions with missing C identifier included: %d\n", debugInfo.MissingCIdentifierIncluded)
	}
	fmt.Println("============================\n")
}

func processGIR(gir *girparser.GIR, filterPattern string, outputHeader string, outputImpl string, verbose bool, outputOnly bool, dumpCIdentifiers bool, ignoreIntrospection bool) {
	if verbose {
		fmt.Printf("Successfully parsed GIR file for %s version %s\n",
			gir.Namespace.Name, gir.Namespace.Version)
	}

	// Dump C identifiers if requested
	if dumpCIdentifiers {
		identifiers := girparser.DumpCIdentifiers(gir)
		fmt.Println("\n=== C Identifiers ===")
		for _, id := range identifiers {
			fmt.Println(id)
		}
		fmt.Println("=====================\n")
	}

	// Set up the filter function for special handling
	var includeFilter func(string) bool
	if filterPattern != "" {
		pattern, err := regexp.Compile(filterPattern)
		if err != nil {
			fmt.Printf("Error compiling regex pattern: %v\n", err)
			os.Exit(1)
		}

		includeFilter = func(name string) bool {
			return pattern.MatchString(name)
		}
	} else if ignoreIntrospection {
		// If ignoreIntrospection is enabled but no filter pattern is specified,
		// allow all functions regardless of introspection status
		includeFilter = func(string) bool { return true }
	}

	// Extract Vips functions
	allFunctions, debugInfo := girparser.GetVipsFunctions(gir, includeFilter)

	if verbose {
		fmt.Printf("Found %d Vips functions\n", len(allFunctions))

		if len(debugInfo.FoundFunctionNames) > 0 {
			fmt.Println("\nFound functions:")
			for i, name := range debugInfo.FoundFunctionNames {
				if i > 20 && len(debugInfo.FoundFunctionNames) > 25 {
					fmt.Printf("... and %d more\n", len(debugInfo.FoundFunctionNames)-i)
					break
				}
				fmt.Println("  " + name)
			}
			fmt.Println()
		}
	}

	// Filter functions if a pattern was specified
	var functions []girparser.VipsFunctionInfo
	if filterPattern != "" {
		pattern, err := regexp.Compile(filterPattern)
		if err != nil {
			fmt.Printf("Error compiling regex pattern: %v\n", err)
			os.Exit(1)
		}

		functions = girparser.FilterVipsFunctions(allFunctions, func(fn girparser.VipsFunctionInfo) bool {
			return pattern.MatchString(fn.Name)
		})

		if verbose {
			fmt.Printf("Filtered to %d functions matching pattern: %s\n", len(functions), filterPattern)

			if len(functions) > 0 {
				fmt.Println("\nFiltered functions:")
				for _, fn := range functions {
					fmt.Println("  " + fn.Name)
				}
				fmt.Println()
			} else {
				fmt.Println("\nWARNING: No functions matched the filter pattern. Output files will be empty.")
				fmt.Println("Available function names to filter on:")
				for i, name := range debugInfo.FoundFunctionNames {
					if i > 20 && len(debugInfo.FoundFunctionNames) > 25 {
						fmt.Printf("... and %d more\n", len(debugInfo.FoundFunctionNames)-i)
						break
					}
					fmt.Println("  " + name)
				}
				fmt.Println()
			}
		}
	} else {
		functions = allFunctions
	}

	// Create a code generator
	generator, err := girparser.NewVipsCodeGenerator()
	if err != nil {
		fmt.Printf("Error creating code generator: %v\n", err)
		os.Exit(1)
	}

	// Generate header
	headerContent, err := generator.GenerateHeader(gir, functions)
	if err != nil {
		fmt.Printf("Error generating header: %v\n", err)
		os.Exit(1)
	}

	// Generate implementation
	implContent, err := generator.GenerateImplementation(gir, functions)
	if err != nil {
		fmt.Printf("Error generating implementation: %v\n", err)
		os.Exit(1)
	}

	// Output the results
	if outputOnly {
		// Output to stdout
		fmt.Println("/* HEADER FILE */")
		fmt.Println(headerContent)
		fmt.Println("\n/* IMPLEMENTATION FILE */")
		fmt.Println(implContent)
	} else {
		// Write header file
		if err := os.WriteFile(outputHeader, []byte(headerContent), 0644); err != nil {
			fmt.Printf("Error writing header file: %v\n", err)
			os.Exit(1)
		}

		if verbose {
			fmt.Printf("Generated header file: %s\n", outputHeader)
		}

		// Write implementation file
		if err := os.WriteFile(outputImpl, []byte(implContent), 0644); err != nil {
			fmt.Printf("Error writing implementation file: %v\n", err)
			os.Exit(1)
		}

		if verbose {
			fmt.Printf("Generated implementation file: %s\n", outputImpl)
		}
	}

	// Print a sample of generated functions
	if verbose && len(functions) > 0 {
		sampleCount := 3
		if len(functions) < sampleCount {
			sampleCount = len(functions)
		}

		fmt.Println("\nSample of generated functions:")
		for i := 0; i < sampleCount; i++ {
			fn := functions[i]
			params := make([]string, len(fn.Params))
			for j, param := range fn.Params {
				params[j] = fmt.Sprintf("%s %s", param.CType, param.Name)
			}

			fmt.Printf("int %s(%s);\n", fn.Name, strings.Join(params, ", "))
		}
	}
}
