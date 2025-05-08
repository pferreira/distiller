package main

import (
    "encoding/json"
    "flag"
    "fmt"
    "go/ast"
    "go/parser"
    "go/token"
    "golang.org/x/net/html"
    "io/ioutil"
    "os"
    "path/filepath"
    "regexp"
    "strings"
    "time"
    "sort"
)

// Variable represents a variable declaration in code
type Variable struct {
    Name  string `json:"name"`
    Type  string `json:"type"`
    Scope string `json:"scope"` // "global", "local", "struct", "property", etc.
    Line  int    `json:"line"`
}

// Function represents a function declaration in code
type Function struct {
    Name     string     `json:"name"`
    Args     []Variable `json:"args"`
    Returns  []string   `json:"returns"`
    Receiver string     `json:"receiver,omitempty"` // For methods
    Line     int        `json:"line"`
    Calls    []string   `json:"calls,omitempty"` // Functions called within this function
}

// ControlFlow represents control flow structures in code
type ControlFlow struct {
    Type     string        `json:"type"` // "if", "for", "switch", "while", "foreach", etc.
    Line     int           `json:"line"`
    Children []ControlFlow `json:"children,omitempty"` // Nested control flow
}

// Struct represents a struct/class definition in code
type Struct struct {
    Name    string     `json:"name"`
    Fields  []Variable `json:"fields"`
    Methods []Function `json:"methods,omitempty"`
    Line    int        `json:"line"`        // Add this field
}

// Interface represents an interface definition in code
type Interface struct {
    Name    string     `json:"name"`
    Methods []Function `json:"methods"`
}

// Import represents an import/include/require statement in code
type Import struct {
    Path string `json:"path"`
}

// GoFileSummary represents a summary of a Go file
type GoFileSummary struct {
    FilePath     string        `json:"filePath"`
    Variables    []Variable    `json:"variables,omitempty"`
    Functions    []Function    `json:"functions,omitempty"`
    ControlFlows []ControlFlow `json:"controlFlows,omitempty"`
    Structs      []Struct      `json:"structs,omitempty"`
    Interfaces   []Interface   `json:"interfaces,omitempty"`
    Imports      []Import      `json:"imports,omitempty"`
}

// PhpFileSummary represents a summary of a PHP file
type PhpFileSummary struct {
    FilePath     string        `json:"filePath"`
    Variables    []Variable    `json:"variables,omitempty"`
    Functions    []Function    `json:"functions,omitempty"`
    ControlFlows []ControlFlow `json:"controlFlows,omitempty"`
    Classes      []Struct      `json:"classes,omitempty"`
    Interfaces   []Interface   `json:"interfaces,omitempty"`
    Imports      []Import      `json:"imports,omitempty"`
}

// PythonFileSummary represents a summary of a Python file
type PythonFileSummary struct {
    FilePath     string        `json:"filePath"`
    Variables    []Variable    `json:"variables,omitempty"`
    Functions    []Function    `json:"functions,omitempty"`
    ControlFlows []ControlFlow `json:"controlFlows,omitempty"`
    Classes      []Struct      `json:"classes,omitempty"`
    Imports      []Import      `json:"imports,omitempty"`
    Decorators   []string      `json:"decorators,omitempty"`
}

// HtmlElement represents an HTML element
type HtmlElement struct {
    ID                string            `json:"id,omitempty"`
    Classes           []string          `json:"classes,omitempty"`
    Attributes        map[string]string `json:"attributes,omitempty"`
    Line              int               `json:"line"`
    LinkedFunctions   []string          `json:"linkedFunctions,omitempty"`
}

// HtmlFileSummary represents a summary of an HTML file
type HtmlFileSummary struct {
    FilePath   string        `json:"filePath"`
    Elements   []HtmlElement `json:"elements"`
    EmbeddedJS []Function    `json:"embeddedJS,omitempty"`
    EmbeddedCSS []CSSRule    `json:"embeddedCSS,omitempty"`
    Includes   []string      `json:"includes,omitempty"`
}

// CSSRule represents a CSS rule
type CSSRule struct {
    Selector string            `json:"selector"`
    Properties map[string]string `json:"properties"`
    Line     int               `json:"line"`
    MediaQuery string          `json:"mediaQuery,omitempty"`
}

// CSSFileSummary represents a summary of a CSS file
type CSSFileSummary struct {
    FilePath string    `json:"filePath"`
    Rules    []CSSRule `json:"rules"`
    Imports  []string  `json:"imports,omitempty"`
}

// SQLStatement represents a SQL statement
type SQLStatement struct {
    Type      string   `json:"type"` // "SELECT", "INSERT", "UPDATE", "DELETE", "CREATE", "ALTER", etc.
    Tables    []string `json:"tables"` 
    Columns   []string `json:"columns,omitempty"`
    Line      int      `json:"line"`
    RawQuery  string   `json:"rawQuery,omitempty"`
}

// SQLFileSummary represents a summary of a SQL file
type SQLFileSummary struct {
    FilePath   string         `json:"filePath"`
    Statements []SQLStatement `json:"statements"`
}

// Summary represents a summary of all analyzed files
type Summary struct {
    GoFiles      []GoFileSummary     `json:"goFiles,omitempty"`
    PhpFiles     []PhpFileSummary    `json:"phpFiles,omitempty"`
    PythonFiles  []PythonFileSummary `json:"pythonFiles,omitempty"`
    HtmlFiles    []HtmlFileSummary   `json:"htmlFiles,omitempty"`
    CssFiles     []CSSFileSummary    `json:"cssFiles,omitempty"`
    SqlFiles     []SQLFileSummary    `json:"sqlFiles,omitempty"`
}

// PatternSummary represents a more concise pattern-based summary format
type PatternSummary struct {
    Timestamp   string           `json:"timestamp"`
    AnalyzedDir string           `json:"analyzedDir"`
    Types       []string         `json:"types,omitempty"`    // All types defined across files
    Functions   []string         `json:"functions,omitempty"` // All function names across files
    FileMap     map[string][]int `json:"fileMap"`           // Maps function/type names to file indices
    Files       []string         `json:"files"`             // All file paths
    CSSSelectors []string        `json:"cssSelectors,omitempty"` // All CSS selectors
    SQLTables   []string         `json:"sqlTables,omitempty"`   // All SQL tables
    Details     Summary          `json:"details"`           // Original full summary
}

// Global variables to store information during parsing
var (
    allFunctions      map[string]Function
    allStructs        map[string]Struct
    allClasses        map[string]Struct
    allPythonClasses  map[string]Struct
    allCSSSelectors   map[string]bool
    allSQLTables      map[string]bool
    currentStructName string
    currentClassName  string
    currentFileName   string
)

// Configuration options
type Config struct {
    Directory       string
    OutputFormat    string // "json" or "pattern"
    Compact         bool
    FilterEmpty     bool
    OnlyRelevant    bool
    MaxResults      int
    TargetFiles     []string
    ExcludePatterns []string
    IncludePatterns []string
    OutputFile      string
    PrintVersion    bool
    Verbose         bool
}

// Version information
const (
    VERSION = "3.0.2"
)

func showHelp() {
    fmt.Println(`Distiller by Philip Ferreira for AI-Assisted Development
Version: ` + VERSION + `

This tool analyzes Go, PHP, Python, HTML, CSS, and SQL files to extract structural information about 
your codebase in a format optimized for AI systems. It's designed to provide an AI with enough 
context to understand code structure without needing the entire codebase.

Usage: distiller [options]

Options:
  -dir string       Directory to analyze (required)
  -files string     Comma-separated list of specific files to analyze
  -exclude string   Comma-separated list of exclude patterns (e.g., "vendor,node_modules,venv")
  -include string   Comma-separated list of include patterns (e.g., "*.go,*.php,*.py,*.html")
  -format string    Output format: "json" or "pattern" (default "json")
  -compact          Output compact JSON without indentation (default true)
  -filter-empty     Filter out empty arrays and slices (default true)
  -relevant         Only include files relevant to target files (default false)
  -max int          Maximum number of files to include (default 0 for all)
  -output string    Output file (default stdout)
  -version          Print version information
  -verbose          Enable verbose output

Examples:
  distiller -dir=./myproject
  distiller -dir=./myproject -files=main.go,index.php,app.py -format=pattern
  distiller -dir=./myproject -exclude=vendor,node_modules,venv -output=summary.json

For bug reporting and feature requests, contact your system administrator.`)
}

func main() {
    // Parse command line arguments
    config := parseFlags()

    // Check if we should just print the version and exit
    if config.PrintVersion {
    fmt.Printf("Multi-Language Code Analyzer v%s\n", VERSION)
    return
    }

    // Validate config
    if config.Directory == "" {
    fmt.Println("Error: Directory is required")
    showHelp()
    os.Exit(1)
    }

    // Start the analyzer
    if config.Verbose {
    fmt.Printf("Analyzing directory: %s\n", config.Directory)
    fmt.Printf("Output format: %s\n", config.OutputFormat)
    fmt.Printf("Compact: %v\n", config.Compact)
    fmt.Printf("Filter empty: %v\n", config.FilterEmpty)
    if len(config.TargetFiles) > 0 {
        fmt.Printf("Target files: %v\n", config.TargetFiles)
    }
    if len(config.ExcludePatterns) > 0 {
        fmt.Printf("Exclude patterns: %v\n", config.ExcludePatterns)
    }
    if len(config.IncludePatterns) > 0 {
        fmt.Printf("Include patterns: %v\n", config.IncludePatterns)
    }
    }

    // Initialize global maps
    allFunctions = make(map[string]Function)
    allStructs = make(map[string]Struct)
    allClasses = make(map[string]Struct)
    allPythonClasses = make(map[string]Struct)
    allCSSSelectors = make(map[string]bool)
    allSQLTables = make(map[string]bool)

    // Add venv to exclude patterns if not already present
    venvExcluded := false
    for _, pattern := range config.ExcludePatterns {
        if pattern == "venv" {
            venvExcluded = true
            break
        }
    }
    if !venvExcluded {
        config.ExcludePatterns = append(config.ExcludePatterns, "venv")
    }
    
    // Analyze the directory
    summary := analyzeDirRecursive(config)

    // Filter empty slices if requested
    if config.FilterEmpty {
    summary = filterEmptySlices(summary)
    }

    // Prepare output based on format
    var outputData []byte
    var err error

    if config.OutputFormat == "pattern" {
    // Convert to pattern format for more efficient AI consumption
    patternSummary := convertToPatternFormat(summary, config)
    if config.Compact {
        outputData, err = json.Marshal(patternSummary)
    } else {
        outputData, err = json.MarshalIndent(patternSummary, "", "  ")
    }
    } else {
    // Use standard JSON format
    if config.Compact {
        outputData, err = json.Marshal(summary)
    } else {
        outputData, err = json.MarshalIndent(summary, "", "  ")
    }
    }

    if err != nil {
    fmt.Printf("Error marshaling JSON: %v\n", err)
    os.Exit(1)
    }

    // Output the result
    if config.OutputFile != "" {
    if config.Verbose {
        fmt.Printf("Writing output to file: %s\n", config.OutputFile)
    }
    err = ioutil.WriteFile(config.OutputFile, outputData, 0644)
    if err != nil {
        fmt.Printf("Error writing to file: %v\n", err)
        os.Exit(1)
    }
    } else {
    fmt.Println(string(outputData))
    }

    if config.Verbose {
    fmt.Printf("Analysis complete. Processed:\n")
    fmt.Printf("- %d Go files\n", len(summary.GoFiles))
    fmt.Printf("- %d PHP files\n", len(summary.PhpFiles))
    fmt.Printf("- %d Python files\n", len(summary.PythonFiles))
    fmt.Printf("- %d HTML files\n", len(summary.HtmlFiles))
    fmt.Printf("- %d CSS files\n", len(summary.CssFiles))
    fmt.Printf("- %d SQL files\n", len(summary.SqlFiles))
    }
}

// parseFlags parses command line flags and returns a Config
func parseFlags() Config {
    config := Config{}

    // Define flags
    flag.StringVar(&config.Directory, "dir", "", "Directory to analyze")
    
    files := flag.String("files", "", "Comma-separated list of specific files to analyze")
    exclude := flag.String("exclude", "", "Comma-separated list of exclude patterns")
    include := flag.String("include", "", "Comma-separated list of include patterns")
    
    flag.StringVar(&config.OutputFormat, "format", "json", "Output format: json or pattern")
    flag.BoolVar(&config.Compact, "compact", true, "Output compact JSON without indentation")
    flag.BoolVar(&config.FilterEmpty, "filter-empty", true, "Filter out empty arrays and slices")
    flag.BoolVar(&config.OnlyRelevant, "relevant", false, "Only include files relevant to target files")
    flag.IntVar(&config.MaxResults, "max", 0, "Maximum number of files to include (0 for all)")
    flag.StringVar(&config.OutputFile, "output", "", "Output file (default stdout)")
    flag.BoolVar(&config.PrintVersion, "version", false, "Print version information")
    flag.BoolVar(&config.Verbose, "verbose", false, "Enable verbose output")

    // Parse the flags
    flag.Parse()

    // Check if help was requested
    if len(os.Args) > 1 && (os.Args[1] == "--help" || os.Args[1] == "-h" || os.Args[1] == "help") {
    showHelp()
    os.Exit(0)
    }

    // Process comma-separated lists
    if *files != "" {
    config.TargetFiles = strings.Split(*files, ",")
    }
    if *exclude != "" {
    config.ExcludePatterns = strings.Split(*exclude, ",")
    }
    if *include != "" {
    config.IncludePatterns = strings.Split(*include, ",")
    }

    return config
}

// analyzeDirRecursive analyzes all relevant files in a directory and its subdirectories
func analyzeDirRecursive(config Config) Summary {
    var summary Summary

    // Prepare file filters
    targetFilesMap := make(map[string]bool)
    for _, f := range config.TargetFiles {
    targetFilesMap[f] = true
    }

    // First pass: collect all functions, structs, classes, etc.
    filepath.Walk(config.Directory, func(path string, info os.FileInfo, err error) error {
    if err != nil {
        if config.Verbose {
	fmt.Printf("Error accessing path %s: %v\n", path, err)
        }
        return nil
    }

    if info.IsDir() {
        // Check if directory should be excluded
        for _, pattern := range config.ExcludePatterns {
	if matched, _ := filepath.Match(pattern, info.Name()); matched {
	    if config.Verbose {
	    fmt.Printf("Skipping directory: %s (matches exclude pattern: %s)\n", info.Name(), pattern)
	    }
	    return filepath.SkipDir
	}
        }
        return nil
    }

    // Check if we should process this file
    shouldProcess := false
    
    // Check if it's one of the target files (if specified)
    if len(targetFilesMap) > 0 {
        _, isTarget := targetFilesMap[info.Name()]
        if isTarget {
	shouldProcess = true
        }
    } else {
        shouldProcess = true
    }
    
    // Apply include/exclude patterns
    for _, pattern := range config.ExcludePatterns {
        if matched, _ := filepath.Match(pattern, info.Name()); matched {
	if config.Verbose {
	    fmt.Printf("Skipping file: %s (matches exclude pattern: %s)\n", info.Name(), pattern)
	}
	shouldProcess = false
	break
        }
    }
    
    if len(config.IncludePatterns) > 0 {
        included := false
        for _, pattern := range config.IncludePatterns {
	if matched, _ := filepath.Match(pattern, info.Name()); matched {
	    included = true
	    break
	}
        }
        shouldProcess = shouldProcess && included
    }

    if !shouldProcess {
        return nil
    }

    relPath, err := filepath.Rel(config.Directory, path)
    if err != nil {
        relPath = path
    }

    // Process different file types
    ext := strings.ToLower(filepath.Ext(path))
    
    switch ext {
    case ".go":
        if config.Verbose {
	fmt.Printf("Analyzing Go file: %s\n", relPath)
        }
        goFile := analyzeGoFile(path)
        summary.GoFiles = append(summary.GoFiles, goFile)

        // Store functions and structs for later reference
        for _, fn := range goFile.Functions {
	allFunctions[fn.Name] = fn
        }
        for _, str := range goFile.Structs {
	allStructs[str.Name] = str
        }
        
    case ".php":
        if config.Verbose {
	fmt.Printf("Analyzing PHP file: %s\n", relPath)
        }
        phpFile := analyzePhpFile(path)
        summary.PhpFiles = append(summary.PhpFiles, phpFile)
        
        // Store functions and classes for later reference
        for _, fn := range phpFile.Functions {
	allFunctions[fn.Name] = fn
        }
        for _, cls := range phpFile.Classes {
	allClasses[cls.Name] = cls
        }

    case ".py":
        if config.Verbose {
            fmt.Printf("Analyzing Python file: %s\n", relPath)
        }
        pyFile := analyzePythonFile(path)
        summary.PythonFiles = append(summary.PythonFiles, pyFile)
        
        // Store functions and classes for later reference
        for _, fn := range pyFile.Functions {
            allFunctions[fn.Name] = fn
        }
        for _, cls := range pyFile.Classes {
            allPythonClasses[cls.Name] = cls
        }
        
    case ".html", ".htm":
        if config.Verbose {
	fmt.Printf("Analyzing HTML file: %s\n", relPath)
        }
        htmlFile := analyzeHtmlFile(path, allFunctions)
        summary.HtmlFiles = append(summary.HtmlFiles, htmlFile)
        
    case ".css":
        if config.Verbose {
	fmt.Printf("Analyzing CSS file: %s\n", relPath)
        }
        cssFile := analyzeCssFile(path)
        summary.CssFiles = append(summary.CssFiles, cssFile)
        
        // Store CSS selectors for later reference
        for _, rule := range cssFile.Rules {
	allCSSSelectors[rule.Selector] = true
        }
        
    case ".sql":
        if config.Verbose {
	fmt.Printf("Analyzing SQL file: %s\n", relPath)
        }
        sqlFile := analyzeSqlFile(path)
        summary.SqlFiles = append(summary.SqlFiles, sqlFile)
        
        // Store SQL tables for later reference
        for _, stmt := range sqlFile.Statements {
	for _, table := range stmt.Tables {
	    allSQLTables[table] = true
	}
        }
    }

    return nil
    })

    // Second pass: establish cross-file relationships and references
    for i := range summary.HtmlFiles {
    for j, element := range summary.HtmlFiles[i].Elements {
        linkedFunctions := findLinkedFunctions(element, allFunctions, allClasses)
        summary.HtmlFiles[i].Elements[j].LinkedFunctions = linkedFunctions
    }
    }

    // Limit results if needed
    if config.MaxResults > 0 {
    if len(summary.GoFiles) > config.MaxResults {
        summary.GoFiles = summary.GoFiles[:config.MaxResults]
    }
    if len(summary.PhpFiles) > config.MaxResults {
        summary.PhpFiles = summary.PhpFiles[:config.MaxResults]
    }
        if len(summary.PythonFiles) > config.MaxResults {
            summary.PythonFiles = summary.PythonFiles[:config.MaxResults]
        }
    if len(summary.HtmlFiles) > config.MaxResults {
        summary.HtmlFiles = summary.HtmlFiles[:config.MaxResults]
    }
    if len(summary.CssFiles) > config.MaxResults {
        summary.CssFiles = summary.CssFiles[:config.MaxResults]
    }
    if len(summary.SqlFiles) > config.MaxResults {
        summary.SqlFiles = summary.SqlFiles[:config.MaxResults]
    }
    }

    return summary
}

// analyzeGoFile analyzes a Go file and returns a GoFileSummary
func analyzeGoFile(filePath string) GoFileSummary {
    currentFileName = filePath
    fset := token.NewFileSet()
    node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
    if err != nil {
    fmt.Printf("Error parsing Go file %s: %v\n", filePath, err)
    return GoFileSummary{FilePath: filePath}
    }

    summary := GoFileSummary{
    FilePath: filePath,
    }

    // Extract imports
    for _, imp := range node.Imports {
    path := strings.Trim(imp.Path.Value, "\"")
    summary.Imports = append(summary.Imports, Import{Path: path})
    }

    // Extract global variables
    for _, decl := range node.Decls {
    if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.VAR {
        for _, spec := range genDecl.Specs {
	if valueSpec, ok := spec.(*ast.ValueSpec); ok {
	    for _, name := range valueSpec.Names {
	    var typeStr string
	    if valueSpec.Type != nil {
	        typeStr = exprToString(valueSpec.Type)
	    } else if len(valueSpec.Values) > 0 {
	        // Infer type from value if possible
	        typeStr = "inferred"
	    }

	    variable := Variable{
	        Name:  name.Name,
	        Type:  typeStr,
	        Scope: "global",
	        Line:  fset.Position(name.Pos()).Line,
	    }
	    summary.Variables = append(summary.Variables, variable)
	    }
	}
        }
    }
    }

    // Extract functions, structs, and interfaces
    ast.Inspect(node, func(n ast.Node) bool {
    switch x := n.(type) {
    case *ast.FuncDecl:
        function := extractFunction(x, fset)
        summary.Functions = append(summary.Functions, function)

        // If this is a method, add it to the struct
        if x.Recv != nil && len(x.Recv.List) > 0 {
	recvType := exprToString(x.Recv.List[0].Type)
	// Remove pointer asterisk if present
	recvType = strings.TrimPrefix(recvType, "*")
	
	// Store the method to add to the struct later
	if s, exists := allStructs[recvType]; exists {
	    s.Methods = append(s.Methods, function)
	    allStructs[recvType] = s
	}
        }

    case *ast.TypeSpec:
        if structType, ok := x.Type.(*ast.StructType); ok {
	currentStructName = x.Name.Name
	structure := Struct{
	    Name:   x.Name.Name,
	    Fields: extractStructFields(structType, fset),
	}
	summary.Structs = append(summary.Structs, structure)
	allStructs[x.Name.Name] = structure

        } else if interfaceType, ok := x.Type.(*ast.InterfaceType); ok {
	intf := Interface{
	    Name:    x.Name.Name,
	    Methods: extractInterfaceMethods(interfaceType, fset),
	}
	summary.Interfaces = append(summary.Interfaces, intf)
        }

    case *ast.IfStmt:
        controlFlow := ControlFlow{
	Type: "if",
	Line: fset.Position(x.If).Line,
        }
        
        // Extract nested control flow
        if x.Body != nil {
	nestedControls := extractNestedControlFlow(x.Body, fset)
	if len(nestedControls) > 0 {
	    controlFlow.Children = nestedControls
	}
        }
        
        summary.ControlFlows = append(summary.ControlFlows, controlFlow)

    case *ast.ForStmt:
        controlFlow := ControlFlow{
	Type: "for",
	Line: fset.Position(x.For).Line,
        }
        
        // Extract nested control flow
        if x.Body != nil {
	nestedControls := extractNestedControlFlow(x.Body, fset)
	if len(nestedControls) > 0 {
	    controlFlow.Children = nestedControls
	}
        }
        
        summary.ControlFlows = append(summary.ControlFlows, controlFlow)
        case *ast.SwitchStmt:
        controlFlow := ControlFlow{
	Type: "switch",
	Line: fset.Position(x.Switch).Line,
        }
        
        // Extract nested control flow from switch cases
        if x.Body != nil {
	for _, stmt := range x.Body.List {
	    if caseClause, ok := stmt.(*ast.CaseClause); ok {
	    for _, caseStmt := range caseClause.Body {
	        if blockStmt, ok := caseStmt.(*ast.BlockStmt); ok {
		nestedControls := extractNestedControlFlow(blockStmt, fset)
		if len(nestedControls) > 0 {
		    controlFlow.Children = append(controlFlow.Children, nestedControls...)
		}
	        }
	    }
	    }
	}
        }
        
        summary.ControlFlows = append(summary.ControlFlows, controlFlow)
    }
    return true
    })

    // Update struct methods
    for i, s := range summary.Structs {
    if updatedStruct, exists := allStructs[s.Name]; exists && len(updatedStruct.Methods) > 0 {
        summary.Structs[i].Methods = updatedStruct.Methods
    }
    }

    return summary
}

// extractNestedControlFlow extracts control flow statements from a block
func extractNestedControlFlow(block *ast.BlockStmt, fset *token.FileSet) []ControlFlow {
    var nestedControls []ControlFlow
    
    for _, stmt := range block.List {
    switch x := stmt.(type) {
    case *ast.IfStmt:
        control := ControlFlow{
	Type: "if",
	Line: fset.Position(x.If).Line,
        }
        
        if x.Body != nil {
	innerNested := extractNestedControlFlow(x.Body, fset)
	if len(innerNested) > 0 {
	    control.Children = innerNested
	}
        }
        
        nestedControls = append(nestedControls, control)
        
    case *ast.ForStmt:
        control := ControlFlow{
	Type: "for",
	Line: fset.Position(x.For).Line,
        }
        
        if x.Body != nil {
	innerNested := extractNestedControlFlow(x.Body, fset)
	if len(innerNested) > 0 {
	    control.Children = innerNested
	}
        }
        
        nestedControls = append(nestedControls, control)
        
    case *ast.SwitchStmt:
        control := ControlFlow{
	Type: "switch",
	Line: fset.Position(x.Switch).Line,
        }
        
        if x.Body != nil {
	for _, switchStmt := range x.Body.List {
	    if caseClause, ok := switchStmt.(*ast.CaseClause); ok {
	    for _, caseStmt := range caseClause.Body {
	        if blockStmt, ok := caseStmt.(*ast.BlockStmt); ok {
		innerNested := extractNestedControlFlow(blockStmt, fset)
		if len(innerNested) > 0 {
		    control.Children = append(control.Children, innerNested...)
		}
	        }
	    }
	    }
	}
        }
        
        nestedControls = append(nestedControls, control)
    }
    }
    
    return nestedControls
}

// extractFunction extracts function details
func extractFunction(funcDecl *ast.FuncDecl, fset *token.FileSet) Function {
    function := Function{
    Name: funcDecl.Name.Name,
    Line: fset.Position(funcDecl.Pos()).Line,
    }

    // Extract receiver for methods
    if funcDecl.Recv != nil && len(funcDecl.Recv.List) > 0 {
    recvType := exprToString(funcDecl.Recv.List[0].Type)
    function.Receiver = strings.TrimPrefix(recvType, "*") // Remove pointer asterisk if present
    }

    // Extract arguments
    if funcDecl.Type.Params != nil {
    for _, field := range funcDecl.Type.Params.List {
        typeStr := exprToString(field.Type)
        for _, name := range field.Names {
	arg := Variable{
	    Name:  name.Name,
	    Type:  typeStr,
	    Scope: "argument",
	    Line:  fset.Position(name.Pos()).Line,
	}
	function.Args = append(function.Args, arg)
        }
    }
    }

    // Extract return types
    if funcDecl.Type.Results != nil {
    for _, field := range funcDecl.Type.Results.List {
        typeStr := exprToString(field.Type)
        function.Returns = append(function.Returns, typeStr)
    }
    }

    // Extract function calls
    if funcDecl.Body != nil {
    ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
        if callExpr, ok := n.(*ast.CallExpr); ok {
	if ident, ok := callExpr.Fun.(*ast.Ident); ok {
	    // Direct function call
	    function.Calls = appendIfNotExists(function.Calls, ident.Name)
	} else if selExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
	    // Method call or package function
	    function.Calls = appendIfNotExists(function.Calls, exprToString(selExpr))
	}
        }
        return true
    })
    }

    return function
}

// extractStructFields extracts fields from a struct definition
func extractStructFields(structType *ast.StructType, fset *token.FileSet) []Variable {
    var fields []Variable

    if structType.Fields == nil {
    return fields
    }

    for _, field := range structType.Fields.List {
    typeStr := exprToString(field.Type)
    
    if len(field.Names) == 0 {
        // Embedded field
        fields = append(fields, Variable{
	Name:  typeStr, // Use the type as the name for embedded fields
	Type:  typeStr,
	Scope: "struct",
	Line:  fset.Position(field.Pos()).Line,
        })
    } else {
        for _, name := range field.Names {
	fields = append(fields, Variable{
	    Name:  name.Name,
	    Type:  typeStr,
	    Scope: "struct",
	    Line:  fset.Position(name.Pos()).Line,
	})
        }
    }
    }

    return fields
}

// extractInterfaceMethods extracts methods from an interface definition
func extractInterfaceMethods(interfaceType *ast.InterfaceType, fset *token.FileSet) []Function {
    var methods []Function

    if interfaceType.Methods == nil {
    return methods
    }

    for _, method := range interfaceType.Methods.List {
    if len(method.Names) > 0 {
        name := method.Names[0].Name
        
        function := Function{
	Name: name,
	Line: fset.Position(method.Pos()).Line,
        }
        
        if funcType, ok := method.Type.(*ast.FuncType); ok {
	// Extract arguments
	if funcType.Params != nil {
	    for _, param := range funcType.Params.List {
	    typeStr := exprToString(param.Type)
	    for _, paramName := range param.Names {
	        arg := Variable{
		Name:  paramName.Name,
		Type:  typeStr,
		Scope: "argument",
		Line:  fset.Position(paramName.Pos()).Line,
	        }
	        function.Args = append(function.Args, arg)
	    }
	    }
	}
	
	// Extract return types
	if funcType.Results != nil {
	    for _, result := range funcType.Results.List {
	    typeStr := exprToString(result.Type)
	    function.Returns = append(function.Returns, typeStr)
	    }
	}
        }
        
        methods = append(methods, function)
    }
    }

    return methods
}

// analyzePhpFile analyzes a PHP file and returns a PhpFileSummary
func analyzePhpFile(filePath string) PhpFileSummary {
    currentFileName = filePath
    
    // Read file content
    data, err := ioutil.ReadFile(filePath)
    if err != nil {
    fmt.Printf("Error reading PHP file %s: %v\n", filePath, err)
    return PhpFileSummary{FilePath: filePath}
    }
    
    content := string(data)
    
    summary := PhpFileSummary{
    FilePath: filePath,
    }
    
    // Parse includes/requires
    includeRegex := regexp.MustCompile(`(?i)(include|require)(_once)?\s*\(\s*['"]([^'"]+)['"]\s*\)`)
    includeMatches := includeRegex.FindAllStringSubmatch(content, -1)
    
    for _, match := range includeMatches {
    if len(match) >= 4 {
        summary.Imports = append(summary.Imports, Import{Path: match[3]})
    }
    }
    
    // Parse classes
    classRegex := regexp.MustCompile(`(?i)class\s+(\w+)(?:\s+extends\s+(\w+))?(?:\s+implements\s+([\w,\s]+))?`)
    classMatches := classRegex.FindAllStringSubmatchIndex(content, -1)
    
    for _, match := range classMatches {
    if len(match) >= 2 {
        startPos := match[0]
        // Find class name
        nameStart := match[2]
        nameEnd := match[3]
        className := content[nameStart:nameEnd]
        currentClassName = className
        
         lineNumber := countLines(content[:startPos])
        
        // This is where the code should go
        class := Struct{
            Name:    className,
            Fields:  extractPhpProperties(content, startPos),
            Methods: extractPhpMethods(content, startPos, className),
            Line:    lineNumber,
        }
        
        // Now extract properties and methods
        summary.Classes = append(summary.Classes, class)
        allClasses[className] = class
    }
    }
    
    // Parse functions
    functionRegex := regexp.MustCompile(`function\s+(\w+)\s*\((.*?)\)`)
    functionMatches := functionRegex.FindAllStringSubmatchIndex(content, -1)
    
    for _, match := range functionMatches {
    if len(match) >= 4 {
        startPos := match[0]
        
        // Skip functions that are part of a class
        if isWithinClass(content, startPos) {
	continue
        }
        
        // Find function name
        nameStart := match[2]
        nameEnd := match[3]
        functionName := content[nameStart:nameEnd]
        
        // Find function arguments
        argsStart := match[4]
        argsEnd := match[5]
        argsStr := content[argsStart:argsEnd]
        
        lineNumber := countLines(content[:startPos])
        
        function := Function{
	Name: functionName,
	Line: lineNumber,
	Args: parsePhpFunctionArgs(argsStr, lineNumber),
        }
        
        // Extract function calls
        function.Calls = extractPhpFunctionCalls(content, startPos)
        
        summary.Functions = append(summary.Functions, function)
        allFunctions[functionName] = function
    }
    }
    
    // Parse control flow
    summary.ControlFlows = extractPhpControlFlow(content)
    
    // Parse global variables
    globalVarRegex := regexp.MustCompile(`\$(\w+)\s*=`)
    globalVarMatches := globalVarRegex.FindAllStringSubmatchIndex(content, -1)
    
    for _, match := range globalVarMatches {
    if len(match) >= 2 {
        startPos := match[0]
        
        // Skip variables that are within functions or classes
        if isWithinFunction(content, startPos) || isWithinClass(content, startPos) {
	continue
        }
        
        // Find variable name
        nameStart := match[2]
        nameEnd := match[3]
        varName := content[nameStart:nameEnd]
        
        lineNumber := countLines(content[:startPos])
        
        variable := Variable{
	Name:  "$" + varName,
	Type:  "inferred",
	Scope: "global",
	Line:  lineNumber,
        }
        
        summary.Variables = append(summary.Variables, variable)
    }
    }
    
    return summary
}

// analyzePythonFile analyzes a Python file and returns a PythonFileSummary
func analyzePythonFile(filePath string) PythonFileSummary {
    currentFileName = filePath
    
    // Read file content
    data, err := ioutil.ReadFile(filePath)
    if err != nil {
        fmt.Printf("Error reading Python file %s: %v\n", filePath, err)
        return PythonFileSummary{FilePath: filePath}
    }
    
    content := string(data)
    
    summary := PythonFileSummary{
        FilePath: filePath,
    }
    
    // Parse imports
    importRegexes := []*regexp.Regexp{
        regexp.MustCompile(`(?m)^import\s+([^#\n]+)`),                 // import module
        regexp.MustCompile(`(?m)^from\s+([^\s]+)\s+import\s+([^#\n]+)`), // from module import ...
    }
    
    for _, regex := range importRegexes {
        matches := regex.FindAllStringSubmatch(content, -1)
        
        for _, match := range matches {
            if len(match) >= 2 {
                importPath := strings.TrimSpace(match[1])
                
                // Handle multiple imports on a single line
                if strings.Contains(importPath, ",") {
                    for _, imp := range strings.Split(importPath, ",") {
                        imp = strings.TrimSpace(imp)
                        if imp != "" {
                            summary.Imports = append(summary.Imports, Import{Path: imp})
                        }
                    }
                } else {
                    summary.Imports = append(summary.Imports, Import{Path: importPath})
                }
                
                // If it's a 'from ... import' statement
                if len(match) >= 3 {
                    imports := strings.TrimSpace(match[2])
                    if imports == "*" {
                        // import all
                        summary.Imports = append(summary.Imports, Import{Path: importPath + ".*"})
                    } else if strings.Contains(imports, ",") {
                        for _, imp := range strings.Split(imports, ",") {
                            imp = strings.TrimSpace(imp)
                            if imp != "" {
                                summary.Imports = append(summary.Imports, Import{Path: importPath + "." + imp})
                            }
                        }
                    } else {
                        summary.Imports = append(summary.Imports, Import{Path: importPath + "." + imports})
                    }
                }
            }
        }
    }
    
    // Parse classes
    classRegex := regexp.MustCompile(`(?m)^class\s+(\w+)(?:\s*\(\s*([^)]*)\s*\))?:`)
    classMatches := classRegex.FindAllStringSubmatchIndex(content, -1)
    
    for _, match := range classMatches {
        if len(match) >= 2 {
            startPos := match[0]
            // Find class name
            nameStart := match[2]
            nameEnd := match[3]
            className := content[nameStart:nameEnd]
            
            // Find parent classes if any
            var parentClasses []string
            if len(match) >= 6 && match[4] != -1 && match[5] != -1 {
                parentsStr := content[match[4]:match[5]]
                parents := strings.Split(parentsStr, ",")
                
                for _, parent := range parents {
                    parent = strings.TrimSpace(parent)
                    if parent != "" {
                        parentClasses = append(parentClasses, parent)
                    }
                }
            }
            
            lineNumber := countLines(content[:startPos])
            
            // Find class body (everything indented after the class declaration)
            classBodyStart := match[1] + 1 // Skip the colon
            
            // Extract class methods and fields
            class := Struct{
                Name:    className,
                Fields:  extractPythonClassFields(content, classBodyStart),
                Methods: extractPythonClassMethods(content, classBodyStart, className),
                Line:    lineNumber,
            }
            
            summary.Classes = append(summary.Classes, class)
            allPythonClasses[className] = class
        }
    }
    
    // Parse functions (outside classes)
    funcRegex := regexp.MustCompile(`(?m)^def\s+(\w+)\s*\(\s*(.*?)\s*\):`)
    funcMatches := funcRegex.FindAllStringSubmatchIndex(content, -1)
    
    for _, match := range funcMatches {
        if len(match) >= 4 {
            startPos := match[0]
            
            // Skip methods (functions inside classes)
            if isPythonMethod(content, startPos) {
                continue
            }
            
            // Find function name
            nameStart := match[2]
            nameEnd := match[3]
            functionName := content[nameStart:nameEnd]
            
            // Find function arguments
            argsStart := match[4]
            argsEnd := match[5]
            argsStr := content[argsStart:argsEnd]
            
            lineNumber := countLines(content[:startPos])
            
            // Extract decorators
            decorators := extractPythonDecorators(content, startPos)
            for _, decorator := range decorators {
                summary.Decorators = appendIfNotExists(summary.Decorators, decorator)
            }
            
            // Create function
            function := Function{
                Name: functionName,
                Line: lineNumber,
                Args: parsePythonFunctionArgs(argsStr, lineNumber),
            }
            
            // Extract return type hints if present
            returnTypeHint := extractPythonReturnType(content, match[1])
            if returnTypeHint != "" {
                function.Returns = append(function.Returns, returnTypeHint)
            }
            
            // Extract function calls
            function.Calls = extractPythonFunctionCalls(content, startPos)
            
            summary.Functions = append(summary.Functions, function)
            allFunctions[functionName] = function
        }
    }
    
    // Parse control flow
    summary.ControlFlows = extractPythonControlFlow(content)
    
    // Parse global variables
    globalVarRegex := regexp.MustCompile(`(?m)^(\w+)\s*=`)
    globalVarMatches := globalVarRegex.FindAllStringSubmatchIndex(content, -1)
    
    for _, match := range globalVarMatches {
        if len(match) >= 2 {
            startPos := match[0]
            
            // Skip variables that are within functions or classes
            if isPythonWithinFunction(content, startPos) || isPythonWithinClass(content, startPos) {
                continue
            }
            
            // Find variable name
            nameStart := match[2]
            nameEnd := match[3]
            varName := content[nameStart:nameEnd]
            
            // Skip if it's an import statement
            if varName == "import" || varName == "from" {
                continue
            }
            
            lineNumber := countLines(content[:startPos])
            
            // Try to infer type from the assignment
            varType := "inferred"
            lineEnd := strings.Index(content[startPos:], "\n")
            if lineEnd == -1 {
                lineEnd = len(content) - startPos
            }
            
            // Check for type hints (varname: Type = value)
            typeHintRegex := regexp.MustCompile(`(\w+)\s*:\s*([^=]+)`)
            typeHintMatch := typeHintRegex.FindStringSubmatch(content[startPos:startPos+lineEnd])
            if len(typeHintMatch) >= 3 {
                varType = strings.TrimSpace(typeHintMatch[2])
            }
            
            variable := Variable{
                Name:  varName,
                Type:  varType,
                Scope: "global",
                Line:  lineNumber,
            }
            
            summary.Variables = append(summary.Variables, variable)
        }
    }
    
    return summary
}

// Helper functions for Python analysis

// extractPythonClassFields extracts class fields (attributes)
func extractPythonClassFields(content string, classBodyStart int) []Variable {
    var fields []Variable
    
    // Find lines that could contain class variables
    fieldRegex := regexp.MustCompile(`(?m)^\s+(\w+)\s*=`)
    fieldMatches := fieldRegex.FindAllStringSubmatchIndex(content[classBodyStart:], -1)
    
    for _, match := range fieldMatches {
        if len(match) >= 2 {
            startPos := classBodyStart + match[0]
            
            // Skip if within a method
            if isPythonWithinMethod(content, startPos) {
                continue
            }
            
            // Find variable name
            nameStart := match[2]
            nameEnd := match[3]
            fieldName := content[classBodyStart+nameStart:classBodyStart+nameEnd]
            
            lineNumber := countLines(content[:startPos])
            
            // Try to determine type from hints if present
            fieldType := "inferred"
            lineEnd := strings.Index(content[startPos:], "\n")
            if lineEnd == -1 {
                lineEnd = len(content) - startPos
            }
            
            typeHintRegex := regexp.MustCompile(`(\w+)\s*:\s*([^=]+)`)
            typeHintMatch := typeHintRegex.FindStringSubmatch(content[startPos:startPos+lineEnd])
            if len(typeHintMatch) >= 3 {
                fieldType = strings.TrimSpace(typeHintMatch[2])
            }
            
            field := Variable{
                Name:  fieldName,
                Type:  fieldType,
                Scope: "class",
                Line:  lineNumber,
            }
            
            fields = append(fields, field)
        }
    }
    
    return fields
}

// extractPythonClassMethods extracts methods from a Python class
func extractPythonClassMethods(content string, classBodyStart int, className string) []Function {
    var methods []Function
    
    // Find method definitions
    methodRegex := regexp.MustCompile(`(?m)^\s+def\s+(\w+)\s*\(\s*(.*?)\s*\):`)
    methodMatches := methodRegex.FindAllStringSubmatchIndex(content[classBodyStart:], -1)
    
    for _, match := range methodMatches {
        if len(match) >= 4 {
            startPos := classBodyStart + match[0]
            
            // Find method name
            nameStart := match[2]
            nameEnd := match[3]
            methodName := content[classBodyStart+nameStart:classBodyStart+nameEnd]
            
            // Find method arguments
            argsStart := match[4]
            argsEnd := match[5]
            argsStr := content[classBodyStart+argsStart:classBodyStart+argsEnd]
            
            lineNumber := countLines(content[:startPos])
            
            // Extract decorators
            decorators := extractPythonDecorators(content, startPos)
            
            // Check if it's a static/class method
            isStatic := false
            isClassMethod := false
            for _, dec := range decorators {
                if dec == "staticmethod" {
                    isStatic = true
                } else if dec == "classmethod" {
                    isClassMethod = true
                }
            }
            
            // Create method
            method := Function{
                Name:     methodName,
                Receiver: className,
                Line:     lineNumber,
                Args:     parsePythonFunctionArgs(argsStr, lineNumber),
            }
            
            // Process 'self' or 'cls' parameter if present
            if len(method.Args) > 0 {
                if isStatic {
                    // Static methods don't have self/cls
                } else if isClassMethod {
                    // Class methods have cls as first parameter
                    if method.Args[0].Name == "cls" {
                        method.Args = method.Args[1:]
                    }
                } else {
                    // Instance methods have self as first parameter
                    if method.Args[0].Name == "self" {
                        method.Args = method.Args[1:]
                    }
                }
            }
            
            // Extract return type hints if present
            returnTypeHint := extractPythonReturnType(content, classBodyStart+match[1])
            if returnTypeHint != "" {
                method.Returns = append(method.Returns, returnTypeHint)
            }
            
            // Extract function calls
            method.Calls = extractPythonFunctionCalls(content, startPos)
            
            methods = append(methods, method)
        }
    }
    
    return methods
}

// parsePythonFunctionArgs parses Python function arguments
func parsePythonFunctionArgs(argsStr string, lineNumber int) []Variable {
    var args []Variable
    
    if argsStr == "" {
        return args
    }
    
    // Split arguments by comma
    argsList := strings.Split(argsStr, ",")
    
    for _, arg := range argsList {
        arg = strings.TrimSpace(arg)
        if arg == "" {
            continue
        }
        
        // Parse parameter with default value
        parts := strings.SplitN(arg, "=", 2)
        paramStr := strings.TrimSpace(parts[0])
        
        // Check for type hints (param: Type)
        typeHintParts := strings.SplitN(paramStr, ":", 2)
        paramName := strings.TrimSpace(typeHintParts[0])
        paramType := "Any"
        
        if len(typeHintParts) > 1 {
            paramType = strings.TrimSpace(typeHintParts[1])
        }
        
        // Skip *args and **kwargs syntax
        if strings.HasPrefix(paramName, "*") {
            continue
        }
        
        args = append(args, Variable{
            Name:  paramName,
            Type:  paramType,
            Scope: "parameter",
            Line:  lineNumber,
        })
    }
    
    return args
}

// extractPythonDecorators extracts decorators for a Python function or method
func extractPythonDecorators(content string, funcPos int) []string {
    var decorators []string
    
    // Look backwards from the function definition to find decorators
    searchStart := 0
    if funcPos > 200 {
        searchStart = funcPos - 200 // Look at most 200 chars back
    }
    
    searchText := content[searchStart:funcPos]
    
    // Find all decorators before this function
    decoratorRegex := regexp.MustCompile(`@(\w+)(?:\(.*?\))?`)
    decoratorMatches := decoratorRegex.FindAllStringSubmatch(searchText, -1)
    
    for _, match := range decoratorMatches {
        if len(match) >= 2 {
            decorators = append(decorators, match[1])
        }
    }
    
    return decorators
}

// extractPythonReturnType extracts the return type hint from a Python function
func extractPythonReturnType(content string, funcEnd int) string {
    // Look for "-> Type:" pattern in function signature
    endOfLine := strings.Index(content[funcEnd:], "\n")
    if endOfLine == -1 {
        endOfLine = len(content) - funcEnd
    }
    
    searchText := content[funcEnd-20:funcEnd+endOfLine] // Include some context before
    
    returnTypeRegex := regexp.MustCompile(`->\s*([^:]+)`)
    returnTypeMatch := returnTypeRegex.FindStringSubmatch(searchText)
    
    if len(returnTypeMatch) >= 2 {
        return strings.TrimSpace(returnTypeMatch[1])
    }
    
    return ""
}

// extractPythonFunctionCalls finds function calls within a Python function
func extractPythonFunctionCalls(content string, funcPos int) []string {
    var calls []string
    
    // Find the function body by detecting indentation
    lines := strings.Split(content[funcPos:], "\n")
    if len(lines) < 2 {
        return calls
    }
    
    // Determine body indentation level from the first non-empty line after the def
    indentLevel := 0
    bodyStartLine := 1
    
    for bodyStartLine < len(lines) {
        line := lines[bodyStartLine]
        if strings.TrimSpace(line) == "" {
            bodyStartLine++
            continue
        }
        
        // Count leading spaces to determine indentation
        indentLevel = len(line) - len(strings.TrimLeft(line, " \t"))
        break
    }
    
    if bodyStartLine >= len(lines) || indentLevel == 0 {
        return calls
    }
    
    // Extract the function body based on indentation
    var bodyLines []string
    
    for i := bodyStartLine; i < len(lines); i++ {
        line := lines[i]
        if strings.TrimSpace(line) == "" {
            continue
        }
        
        currentIndent := len(line) - len(strings.TrimLeft(line, " \t"))
        if currentIndent < indentLevel {
            break // End of function body
        }
        
        bodyLines = append(bodyLines, line)
    }
    
    // Join body lines and find function calls
    bodyText := strings.Join(bodyLines, "\n")
    
    // Find direct function calls (name(...))
    callRegex := regexp.MustCompile(`(\w+)\s*\(`)
    callMatches := callRegex.FindAllStringSubmatch(bodyText, -1)
    
    for _, match := range callMatches {
        if len(match) >= 2 {
            funcName := match[1]
            
            // Skip Python built-ins and keywords
            if isPythonKeywordOrBuiltin(funcName) {
                continue
            }
            
            calls = appendIfNotExists(calls, funcName)
        }
    }
    
    // Find method calls (obj.method(...))
    methodCallRegex := regexp.MustCompile(`(\w+)\.(\w+)\s*\(`)
    methodCallMatches := methodCallRegex.FindAllStringSubmatch(bodyText, -1)
    
    for _, match := range methodCallMatches {
        if len(match) >= 3 {
            objName := match[1]
            methodName := match[2]
            
            // Skip Python built-ins
            if isPythonKeywordOrBuiltin(methodName) {
                continue
            }
            
            calls = appendIfNotExists(calls, objName+"."+methodName)
        }
    }
    
    return calls
}

// extractPythonControlFlow finds control flow structures in Python code
func extractPythonControlFlow(content string) []ControlFlow {
    var controls []ControlFlow
    
    // Define regex patterns for control structures
    patterns := map[string]*regexp.Regexp{
        "if":      regexp.MustCompile(`(?m)^(\s*)if\s+.+:`),
        "for":     regexp.MustCompile(`(?m)^(\s*)for\s+.+:`),
        "while":   regexp.MustCompile(`(?m)^(\s*)while\s+.+:`),
        "try":     regexp.MustCompile(`(?m)^(\s*)try\s*:`),
        "with":    regexp.MustCompile(`(?m)^(\s*)with\s+.+:`),
    }
    
    for controlType, pattern := range patterns {
        matches := pattern.FindAllStringSubmatchIndex(content, -1)
        
        for _, match := range matches {
            if len(match) >= 2 {
                startPos := match[0]
                lineNumber := countLines(content[:startPos])
                
                // Determine indentation level to find nested structures
                indentStart := match[2]
                indentEnd := match[3]
                indentation := content[indentStart:indentEnd]
                
                control := ControlFlow{
                    Type: controlType,
                    Line: lineNumber,
                }
                
                // Find nested control structures
                children := findNestedPythonControlFlow(content, startPos, len(indentation))
                if len(children) > 0 {
                    control.Children = children
                }
                
                controls = append(controls, control)
            }
        }
    }
    
    return controls
}

// findNestedPythonControlFlow identifies nested control structures in Python
func findNestedPythonControlFlow(content string, startPos int, parentIndent int) []ControlFlow {
    var nested []ControlFlow
    
    // Define regex patterns for control structures
    patterns := map[string]*regexp.Regexp{
        "if":      regexp.MustCompile(`(?m)^(\s*)if\s+.+:`),
        "for":     regexp.MustCompile(`(?m)^(\s*)for\s+.+:`),
        "while":   regexp.MustCompile(`(?m)^(\s*)while\s+.+:`),
        "try":     regexp.MustCompile(`(?m)^(\s*)try\s*:`),
        "with":    regexp.MustCompile(`(?m)^(\s*)with\s+.+:`),
    }
    
    // Find end of parent block
    endPos := len(content)
    lines := strings.Split(content[startPos:], "\n")
    if len(lines) > 1 {
        // Skip the first line (current control statement)
        linePos := startPos + len(lines[0]) + 1
        for i := 1; i < len(lines); i++ {
            line := lines[i]
            if strings.TrimSpace(line) == "" {
                linePos += len(line) + 1
                continue
            }
            
            currentIndent := len(line) - len(strings.TrimLeft(line, " \t"))
            if currentIndent <= parentIndent {
                endPos = linePos
                break // End of parent block
            }
            
            linePos += len(line) + 1
        }
    }
    
    // Make sure endPos is within bounds
    if endPos > len(content) {
        endPos = len(content)
    }
    
    // Make sure startPos is within bounds and startPos <= endPos
    if startPos >= len(content) {
        return nested
    }
    
    if startPos > endPos {
        return nested
    }
    
    // Look for nested control structures within the parent block
    blockContent := content[startPos:endPos]
    
    for controlType, pattern := range patterns {
        matches := pattern.FindAllStringSubmatchIndex(blockContent, -1)
        
        for _, match := range matches {
            if len(match) >= 4 {
                nestedStartPos := startPos + match[0]
                
                // Get indentation of nested control
                nestedIndentStart := match[2]
                nestedIndentEnd := match[3]
                
                // Make sure indices are within bounds
                if nestedIndentStart < 0 || nestedIndentEnd > len(blockContent) || nestedIndentStart > nestedIndentEnd {
                    continue
                }
                
                nestedIndent := len(blockContent[nestedIndentStart:nestedIndentEnd])
                
                // Ensure it's actually nested (more indented than parent)
                if nestedIndent <= parentIndent {
                    continue
                }
                
                lineNumber := countLines(content[:nestedStartPos])
                
                control := ControlFlow{
                    Type: controlType,
                    Line: lineNumber,
                }
                
                // Find nested control flow (recursively)
                children := findNestedPythonControlFlow(content, nestedStartPos, nestedIndent)
                if len(children) > 0 {
                    control.Children = children
                }
                
                nested = append(nested, control)
            }
        }
    }
    
    return nested
}

// isPythonMethod checks if a function definition is inside a class
func isPythonMethod(content string, pos int) bool {
    // Get the line containing the function definition
    lineStart := strings.LastIndex(content[:pos], "\n")
    if lineStart == -1 {
        lineStart = 0
    } else {
        lineStart++
    }
    
    line := content[lineStart:pos]
    
    // Count indentation
    indentLevel := len(line) - len(strings.TrimLeft(line, " \t"))
    
    // Functions at root level have no indentation
    return indentLevel > 0
}

// isPythonWithinFunction checks if a position is inside a function
// isPythonWithinFunction checks if a position is inside a function
func isPythonWithinFunction(content string, pos int) bool {
    // Look backwards for the nearest function definition at the same or lower indentation
    lineStart := strings.LastIndex(content[:pos], "\n")
    if lineStart == -1 {
        lineStart = 0
    } else {
        lineStart++
    }
    
    currentLine := content[lineStart:pos]
    currentIndent := len(currentLine) - len(strings.TrimLeft(currentLine, " \t"))
    
    // Look back through the file to find the nearest def or class
    lines := strings.Split(content[:lineStart], "\n")
    
    for i := len(lines) - 1; i >= 0; i-- {
        line := lines[i]
        if strings.TrimSpace(line) == "" {
            continue
        }
        
        lineIndent := len(line) - len(strings.TrimLeft(line, " \t"))
        
        // If we find a line with less or equal indentation...
        if lineIndent <= currentIndent {
            // If it's a function definition, we're inside a function
            trimmedLine := strings.TrimSpace(line)
            if len(trimmedLine) >= 3 && trimmedLine[:3] == "def" {
                return true
            }
            
            // If we hit another block with less indentation, we're not in a function
            if lineIndent < currentIndent && len(trimmedLine) > 0 && strings.HasSuffix(trimmedLine, ":") {
                return false
            }
        }
    }
    
    return false
}

// isPythonWithinClass checks if a position is inside a class
func isPythonWithinClass(content string, pos int) bool {
    // Look backwards for the nearest class definition at the same or lower indentation
    lineStart := strings.LastIndex(content[:pos], "\n")
    if lineStart == -1 {
        lineStart = 0
    } else {
        lineStart++
    }
    
    currentLine := content[lineStart:pos]
    currentIndent := len(currentLine) - len(strings.TrimLeft(currentLine, " \t"))
    
    // Look back through the file to find the nearest def or class
    lines := strings.Split(content[:lineStart], "\n")
    
    for i := len(lines) - 1; i >= 0; i-- {
        line := lines[i]
        if strings.TrimSpace(line) == "" {
            continue
        }
        
        lineIndent := len(line) - len(strings.TrimLeft(line, " \t"))
        
        // If we find a line with less or equal indentation...
        if lineIndent < currentIndent {
            // If it's a class definition, we're inside a class
            if strings.TrimSpace(line)[:5] == "class" {
                return true
            }
            
            // If we hit another block with less indentation, we're not in a class
            if strings.HasSuffix(strings.TrimSpace(line), ":") {
                return false
            }
        }
    }
    
    return false
}

// isPythonWithinMethod checks if a position is inside a method
func isPythonWithinMethod(content string, pos int) bool {
    // First check if we're inside a function
    if !isPythonWithinFunction(content, pos) {
        return false
    }
    
    // Now check if we're inside a class
    return isPythonWithinClass(content, pos)
}

// isPythonKeywordOrBuiltin checks if a name is a Python keyword or builtin
func isPythonKeywordOrBuiltin(name string) bool {
    keywords := map[string]bool{
        "False": true, "None": true, "True": true, "and": true, "as": true,
        "assert": true, "async": true, "await": true, "break": true, "class": true,
        "continue": true, "def": true, "del": true, "elif": true, "else": true,
        "except": true, "finally": true, "for": true, "from": true, "global": true,
        "if": true, "import": true, "in": true, "is": true, "lambda": true,
        "nonlocal": true, "not": true, "or": true, "pass": true, "raise": true,
        "return": true, "try": true, "while": true, "with": true, "yield": true,
    }
    
    builtins := map[string]bool{
        "abs": true, "all": true, "any": true, "bin": true, "bool": true,
        "chr": true, "dict": true, "dir": true, "divmod": true, "enumerate": true,
        "filter": true, "float": true, "format": true, "frozenset": true,
        "getattr": true, "hasattr": true, "hash": true, "hex": true, "id": true,
        "input": true, "int": true, "isinstance": true, "issubclass": true,
        "iter": true, "len": true, "list": true, "map": true, "max": true,
        "min": true, "next": true, "oct": true, "open": true, "ord": true,
        "pow": true, "print": true, "property": true, "range": true, "repr": true,
        "reversed": true, "round": true, "set": true, "slice": true, "sorted": true,
        "staticmethod": true, "str": true, "sum": true, "super": true, "tuple": true,
        "type": true, "vars": true, "zip": true,
    }
    
    return keywords[name] || builtins[name]
}

// Helper functions for PHP analysis

// extractPhpProperties finds class properties in PHP code
func extractPhpProperties(content string, classStartPos int) []Variable {
    var properties []Variable
    
    // Find the class body
    openBracePos := strings.Index(content[classStartPos:], "{")
    if openBracePos == -1 {
    return properties
    }
    
    classBodyStart := classStartPos + openBracePos + 1
    
    // Find property declarations (public, protected, private)
    propertyRegex := regexp.MustCompile(`(?i)(public|protected|private)\s+(\$\w+)`)
    propertyMatches := propertyRegex.FindAllStringSubmatchIndex(content[classBodyStart:], -1)
    
    for _, match := range propertyMatches {
    if len(match) >= 4 {
        propPos := classBodyStart + match[0]
        
        // Skip if this is within a method
        if isWithinMethod(content, propPos) {
	continue
        }
        
        // Find visibility
        visibilityStart := match[2]
        visibilityEnd := match[3]
        visibility := content[classBodyStart+visibilityStart:classBodyStart+visibilityEnd]
        
        // Find property name
        nameStart := match[4]
        nameEnd := match[5]
        propName := content[classBodyStart+nameStart:classBodyStart+nameEnd]
        
        lineNumber := countLines(content[:propPos])
        
        property := Variable{
	Name:  propName,
	Type:  "inferred",
	Scope: visibility,
	Line:  lineNumber,
        }
        
        properties = append(properties, property)
    }
    }
    
    return properties
}

// exprToString converts an ast.Expr to a string representation
func exprToString(expr ast.Expr) string {
    switch t := expr.(type) {
    case *ast.Ident:
        return t.Name
    case *ast.SelectorExpr:
        return exprToString(t.X) + "." + t.Sel.Name
    case *ast.StarExpr:
        return "*" + exprToString(t.X)
    case *ast.ArrayType:
        if t.Len == nil {
            return "[]" + exprToString(t.Elt)
        }
        return "[" + exprToString(t.Len) + "]" + exprToString(t.Elt)
    case *ast.BasicLit:
        return t.Value
    case *ast.MapType:
        return "map[" + exprToString(t.Key) + "]" + exprToString(t.Value)
    case *ast.InterfaceType:
        return "interface{}"
    case *ast.FuncType:
        return "func"
    case *ast.ChanType:
        return "chan " + exprToString(t.Value)
    case *ast.StructType:
        return "struct{}"
    case *ast.Ellipsis:
        return "..." + exprToString(t.Elt)
    default:
        return fmt.Sprintf("<%T>", expr)
    }
}

// extractPhpMethods finds methods in a PHP class
func extractPhpMethods(content string, classStartPos int, className string) []Function {
    var methods []Function
    
    // Find the class body
    openBracePos := strings.Index(content[classStartPos:], "{")
    if openBracePos == -1 {
    return methods
    }
    
    classBodyStart := classStartPos + openBracePos + 1
    
    // Find method declarations
    methodRegex := regexp.MustCompile(`(?i)(public|protected|private)?\s*function\s+(\w+)\s*\((.*?)\)`)
    methodMatches := methodRegex.FindAllStringSubmatchIndex(content[classBodyStart:], -1)
    
    for _, match := range methodMatches {
    if len(match) >= 6 {
        methodPos := classBodyStart + match[0]
        
        // Find method name
        nameStart := match[4]
        nameEnd := match[5]
        methodName := content[classBodyStart+nameStart:classBodyStart+nameEnd]
        
        // Find args
        argsStart := match[6]
        argsEnd := match[7]
        argsStr := content[classBodyStart+argsStart:classBodyStart+argsEnd]
        
        lineNumber := countLines(content[:methodPos])
        
        method := Function{
	Name:     methodName,
	Receiver: className,
	Line:     lineNumber,
	Args:     parsePhpFunctionArgs(argsStr, lineNumber),
        }
        
        // Extract function calls
        method.Calls = extractPhpFunctionCalls(content, methodPos)
        
        methods = append(methods, method)
    }
    }
    
    return methods
}

// parsePhpFunctionArgs parses PHP function arguments
func parsePhpFunctionArgs(argsStr string, lineNumber int) []Variable {
    var args []Variable
    
    // Split arguments by comma
    argsSplit := strings.Split(argsStr, ",")
    
    for _, arg := range argsSplit {
    arg = strings.TrimSpace(arg)
    if arg == "" {
        continue
    }
    
    // Check for type hint
    typeParts := strings.Fields(arg)
    
    if len(typeParts) >= 2 && strings.HasPrefix(typeParts[len(typeParts)-1], "$") {
        // Has type hint
        varName := typeParts[len(typeParts)-1]
        typeHint := strings.Join(typeParts[:len(typeParts)-1], " ")
        
        args = append(args, Variable{
	Name:  varName,
	Type:  typeHint,
	Scope: "parameter",
	Line:  lineNumber,
        })
    } else if strings.HasPrefix(arg, "$") {
        // No type hint
        args = append(args, Variable{
	Name:  arg,
	Type:  "mixed",
	Scope: "parameter",
	Line:  lineNumber,
        })
    }
    }
    
    return args
}

// extractPhpFunctionCalls finds function calls within a PHP function
func extractPhpFunctionCalls(content string, funcStartPos int) []string {
    var calls []string
    
    // Find the function body
    openBracePos := strings.Index(content[funcStartPos:], "{")
    if openBracePos == -1 {
    return calls
    }
    
    funcBodyStart := funcStartPos + openBracePos + 1
    
    // Find the end of the function
    braceCount := 1
    funcBodyEnd := funcBodyStart
    
    for i := funcBodyStart; i < len(content) && braceCount > 0; i++ {
    if content[i] == '{' {
        braceCount++
    } else if content[i] == '}' {
        braceCount--
    }
    funcBodyEnd = i
    }
    
    if funcBodyEnd <= funcBodyStart {
    return calls
    }
    
    funcBody := content[funcBodyStart:funcBodyEnd]
    
    // Find function calls
    callRegex := regexp.MustCompile(`(\$\w+->)?(\w+)\s*\(`)
    callMatches := callRegex.FindAllStringSubmatch(funcBody, -1)
    
    for _, match := range callMatches {
    if len(match) >= 3 {
        callName := match[2]
        // Skip language constructs
        if callName == "if" || callName == "for" || callName == "while" || callName == "foreach" || callName == "switch" {
	continue
        }
        
        calls = appendIfNotExists(calls, callName)
    }
    }
    
    return calls
}

// extractPhpControlFlow finds control flow structures in PHP code
func extractPhpControlFlow(content string) []ControlFlow {
    var controls []ControlFlow
    
    // Define regex patterns for control structures
    patterns := map[string]*regexp.Regexp{
    "if":      regexp.MustCompile(`if\s*\(`),
    "for":     regexp.MustCompile(`for\s*\(`),
    "while":   regexp.MustCompile(`while\s*\(`),
    "foreach": regexp.MustCompile(`foreach\s*\(`),
    "switch":  regexp.MustCompile(`switch\s*\(`),
    }
    
    for controlType, pattern := range patterns {
    matches := pattern.FindAllStringIndex(content, -1)
    
    for _, match := range matches {
        startPos := match[0]
        
        // Skip if this is a string literal
        if isWithinString(content, startPos) {
	continue
        }
        
        lineNumber := countLines(content[:startPos])
        
        control := ControlFlow{
	Type: controlType,
	Line: lineNumber,
        }
        
        // Find nested control flow
        children := findNestedPhpControlFlow(content, startPos)
        if len(children) > 0 {
	control.Children = children
        }
        
        controls = append(controls, control)
    }
    }
    
    return controls
}

// findNestedPhpControlFlow identifies nested control structures in PHP
func findNestedPhpControlFlow(content string, startPos int) []ControlFlow {
    var nested []ControlFlow
    
    // Find the body of this control structure
    openBracePos := strings.Index(content[startPos:], "{")
    if openBracePos == -1 {
    return nested
    }
    
    bodyStart := startPos + openBracePos + 1
    
    // Find the end of the body
    braceCount := 1
    bodyEnd := bodyStart
    
    for i := bodyStart; i < len(content) && braceCount > 0; i++ {
    if content[i] == '{' {
        braceCount++
    } else if content[i] == '}' {
        braceCount--
    }
    bodyEnd = i
    }
    
    if bodyEnd <= bodyStart {
    return nested
    }
    
    body := content[bodyStart:bodyEnd]
    
    // Search for control structures in this body
    patterns := map[string]*regexp.Regexp{
    "if":      regexp.MustCompile(`if\s*\(`),
    "for":     regexp.MustCompile(`for\s*\(`),
    "while":   regexp.MustCompile(`while\s*\(`),
    "foreach": regexp.MustCompile(`foreach\s*\(`),
    "switch":  regexp.MustCompile(`switch\s*\(`),
    }
    
    for controlType, pattern := range patterns {
    matches := pattern.FindAllStringIndex(body, -1)
    
    for _, match := range matches {
        nestedStartPos := bodyStart + match[0]
        
        // Skip if this is a string literal
        if isWithinString(content, nestedStartPos) {
	continue
        }
        
        lineNumber := countLines(content[:nestedStartPos])
        
        control := ControlFlow{
	Type: controlType,
	Line: lineNumber,
        }
        
        // Find nested control flow (recursively)
        children := findNestedPhpControlFlow(content, nestedStartPos)
        if len(children) > 0 {
	control.Children = children
        }
        
        nested = append(nested, control)
    }
    }
    
    return nested
}

// analyzeHtmlFile analyzes an HTML file with enhanced features
func analyzeHtmlFile(filePath string, allFunctions map[string]Function) HtmlFileSummary {
    data, err := ioutil.ReadFile(filePath)
    if err != nil {
    fmt.Printf("Error reading HTML file %s: %v\n", filePath, err)
    return HtmlFileSummary{FilePath: filePath}
    }

    content := string(data)
    doc, err := html.Parse(strings.NewReader(content))
    if err != nil {
    fmt.Printf("Error parsing HTML file %s: %v\n", filePath, err)
    return HtmlFileSummary{FilePath: filePath}
    }

    summary := HtmlFileSummary{
    FilePath: filePath,
    }

    // Extract includes (PHP includes in HTML)
    includeRegex := regexp.MustCompile(`(?i)<\?(?:php)?\s+(?:include|require)(?:_once)?\s*\(\s*['"]([^'"]+)['"]\s*\)\s*;?\s*\?>`)
    includeMatches := includeRegex.FindAllStringSubmatch(content, -1)
    
    for _, match := range includeMatches {
    if len(match) >= 2 {
        summary.Includes = append(summary.Includes, match[1])
    }
    }

    // Extract embedded JavaScript
    scriptRegex := regexp.MustCompile(`(?s)<script[^>]*>(.*?)</script>`)
    scriptMatches := scriptRegex.FindAllStringSubmatch(content, -1)
    
    for _, match := range scriptMatches {
    if len(match) >= 2 {
        jsContent := match[1]
        if jsContent != "" {
	// Simple function extraction from JS
	funcRegex := regexp.MustCompile(`function\s+(\w+)\s*\((.*?)\)`)
	funcMatches := funcRegex.FindAllStringSubmatch(jsContent, -1)
	
	for _, fMatch := range funcMatches {
	    if len(fMatch) >= 3 {
	    funcName := fMatch[1]
	    lineNum := countLines(content[:strings.Index(content, match[0])]) + 
		   countLines(jsContent[:strings.Index(jsContent, fMatch[0])])
	    
	    function := Function{
	        Name: funcName,
	        Line: lineNum,
	    }
	    
	    summary.EmbeddedJS = append(summary.EmbeddedJS, function)
	    }
	}
        }
    }
    }

    // Extract embedded CSS
    styleRegex := regexp.MustCompile(`(?s)<style[^>]*>(.*?)</style>`)
    styleMatches := styleRegex.FindAllStringSubmatch(content, -1)
    
    for _, match := range styleMatches {
    if len(match) >= 2 {
        cssContent := match[1]
        if cssContent != "" {
	rules := parseCssContent(cssContent)
	
	// Adjust line numbers relative to the HTML file
	baseLineNum := countLines(content[:strings.Index(content, match[0])])
	for i := range rules {
	    rules[i].Line += baseLineNum
	}
	
	summary.EmbeddedCSS = append(summary.EmbeddedCSS, rules...)
        }
    }
    }

    // Process HTML elements
    var processNode func(*html.Node, int) int
    processNode = func(n *html.Node, currentLine int) int {
    if n.Type == html.ElementNode {
        element := HtmlElement{
	Attributes: make(map[string]string),
	Line:       currentLine,
        }

        for _, attr := range n.Attr {
	if attr.Key == "id" {
	    element.ID = attr.Val
	} else if attr.Key == "class" {
	    element.Classes = strings.Fields(attr.Val)
	} else {
	    element.Attributes[attr.Key] = attr.Val
	}
        }

        summary.Elements = append(summary.Elements, element)
    }

    // Estimate line number based on position in the HTML
    for c := n.FirstChild; c != nil; c = c.NextSibling {
        currentLine++
        currentLine = processNode(c, currentLine)
    }

    return currentLine
    }

    processNode(doc, 1)

    return summary
}

// analyzeCssFile analyzes a CSS file
func analyzeCssFile(filePath string) CSSFileSummary {
    data, err := ioutil.ReadFile(filePath)
    if err != nil {
    fmt.Printf("Error reading CSS file %s: %v\n", filePath, err)
    return CSSFileSummary{FilePath: filePath}
    }

    content := string(data)
    
    summary := CSSFileSummary{
    FilePath: filePath,
    }
    
    // Extract @import statements
    importRegex := regexp.MustCompile(`@import\s+(?:url\()?['"]?([^'")]+)['"]?(?:\))?`)
    importMatches := importRegex.FindAllStringSubmatch(content, -1)
    
    for _, match := range importMatches {
    if len(match) >= 2 {
        summary.Imports = append(summary.Imports, match[1])
    }
    }
    
    // Parse CSS rules
    summary.Rules = parseCssContent(content)
    
    return summary
}

// parseCssContent extracts CSS rules from content
func parseCssContent(content string) []CSSRule {
    var rules []CSSRule
    
    // This is a simplified CSS parser - a real implementation would use a proper CSS parser
    // But for the purpose of this example, we'll use regex
    
    // Extract rules with their selectors and content
    ruleRegex := regexp.MustCompile(`([^{]+)(?:{([^}]*)})`)
    ruleMatches := ruleRegex.FindAllStringSubmatch(content, -1)
    
    currentMediaQuery := ""
    
    for _, match := range ruleMatches {
    if len(match) >= 3 {
        selector := strings.TrimSpace(match[1])
        body := match[2]
        
        // Check if this is a media query
        if strings.HasPrefix(selector, "@media") {
	currentMediaQuery = selector
	continue
        }
        
        // Count line number
        lineNum := countLines(content[:strings.Index(content, match[0])])
        
        rule := CSSRule{
	Selector:   selector,
	Properties: make(map[string]string),
	Line:       lineNum,
        }
        
        if currentMediaQuery != "" {
	rule.MediaQuery = currentMediaQuery
        }
        
        // Extract properties
        propRegex := regexp.MustCompile(`([\w-]+)\s*:\s*([^;]+)`)
        propMatches := propRegex.FindAllStringSubmatch(body, -1)
        
        for _, propMatch := range propMatches {
	if len(propMatch) >= 3 {
	    propName := strings.TrimSpace(propMatch[1])
	    propValue := strings.TrimSpace(propMatch[2])
	    rule.Properties[propName] = propValue
	}
        }
        
        rules = append(rules, rule)
        
        // Remember this selector for later
        allCSSSelectors[selector] = true
    }
    }
    
    return rules
}

// analyzeSqlFile analyzes a SQL file
func analyzeSqlFile(filePath string) SQLFileSummary {
    data, err := ioutil.ReadFile(filePath)
    if err != nil {
    fmt.Printf("Error reading SQL file %s: %v\n", filePath, err)
    return SQLFileSummary{FilePath: filePath}
    }

    content := string(data)
    
    summary := SQLFileSummary{
    FilePath: filePath,
    }
    
    // Split into separate SQL statements
    statements := splitSqlStatements(content)
    
    lineNum := 1
    for _, stmt := range statements {
    sqlStmt := parseSqlStatement(stmt, lineNum)
    if sqlStmt.Type != "" {
        summary.Statements = append(summary.Statements, sqlStmt)
        
        // Remember table names for later
        for _, table := range sqlStmt.Tables {
	allSQLTables[table] = true
        }
    }
    
    // Update line number
    lineNum += countLines(stmt)
    }
    
    return summary
}

// splitSqlStatements splits SQL content into separate statements
func splitSqlStatements(content string) []string {
    var statements []string
    
    // This is a simplified SQL splitter - a real implementation would handle more edge cases
    // Like quoted semicolons, multi-line comments, etc.
    
    // Remove comments
    content = removeSqlComments(content)
    
    // Split by semicolons
    parts := strings.Split(content, ";")
    
    for _, part := range parts {
    part = strings.TrimSpace(part)
    if part != "" {
        statements = append(statements, part+";")
    }
    }
    
    return statements
}

// removeSqlComments removes SQL comments from content
func removeSqlComments(content string) string {
    // Remove single-line comments
    singleLineRegex := regexp.MustCompile(`--.*?(\r\n|\r|\n|$)`)
    content = singleLineRegex.ReplaceAllString(content, "$1")
    
    // Remove multi-line comments
    multiLineRegex := regexp.MustCompile(`(?s)/\*.*?\*/`)
    content = multiLineRegex.ReplaceAllString(content, "")
    
    return content
}

// parseSqlStatement analyzes a single SQL statement
func parseSqlStatement(stmt string, lineNum int) SQLStatement {
    sqlStmt := SQLStatement{
    Line:     lineNum,
    RawQuery: stmt,
    }
    
    // Determine statement type
    stmt = strings.TrimSpace(stmt)
    lowerStmt := strings.ToLower(stmt)
    
    if strings.HasPrefix(lowerStmt, "select") {
    sqlStmt.Type = "SELECT"
    sqlStmt.Tables = extractSqlTables(stmt, "from")
    sqlStmt.Columns = extractSqlColumns(stmt)
    } else if strings.HasPrefix(lowerStmt, "insert") {
    sqlStmt.Type = "INSERT"
    sqlStmt.Tables = extractSqlTables(stmt, "into")
    sqlStmt.Columns = extractSqlInsertColumns(stmt)
    } else if strings.HasPrefix(lowerStmt, "update") {
    sqlStmt.Type = "UPDATE"
    sqlStmt.Tables = extractSqlTables(stmt, "update")
    sqlStmt.Columns = extractSqlUpdateColumns(stmt)
    } else if strings.HasPrefix(lowerStmt, "delete") {
    sqlStmt.Type = "DELETE"
    sqlStmt.Tables = extractSqlTables(stmt, "from")
    } else if strings.HasPrefix(lowerStmt, "create table") {
    sqlStmt.Type = "CREATE"
    sqlStmt.Tables = extractSqlTables(stmt, "table")
    sqlStmt.Columns = extractSqlCreateColumns(stmt)
    } else if strings.HasPrefix(lowerStmt, "alter table") {
    sqlStmt.Type = "ALTER"
    sqlStmt.Tables = extractSqlTables(stmt, "table")
    } else {
    // Other statement types (DROP, TRUNCATE, etc.)
    firstWord := strings.Fields(lowerStmt)[0]
    sqlStmt.Type = strings.ToUpper(firstWord)
    
    // Try to find tables
if strings.Contains(lowerStmt, "table") {
        sqlStmt.Tables = extractSqlTables(stmt, "table")
    }
    }
    
    return sqlStmt
}

// Helper functions for SQL analysis
func extractSqlTables(stmt string, keyword string) []string {
    var tables []string
    
    lowerStmt := strings.ToLower(stmt)
    keywordPos := strings.Index(lowerStmt, keyword)
    
    if keywordPos == -1 {
    return tables
    }
    
    // Extract text after the keyword
    restStmt := stmt[keywordPos+len(keyword):]
    
    // Extract table names - this is a simplification
    // A real implementation would handle JOIN clauses, aliases, etc.
    tableRegex := regexp.MustCompile(`[a-zA-Z0-9_]+`)
    tableMatches := tableRegex.FindAllString(restStmt, -1)
    
    // Filter out SQL keywords and keep only table names
    sqlKeywords := map[string]bool{
    "select": true, "where": true, "group": true, "order": true, "by": true,
    "having": true, "limit": true, "offset": true, "join": true, "inner": true,
    "outer": true, "left": true, "right": true, "full": true, "on": true,
    "as": true, "union": true, "except": true, "intersect": true, "values": true,
    "set": true, "into": true, "default": true, "if": true, "exists": true,
    "primary": true, "key": true, "foreign": true, "references": true,
    }
    
    for _, match := range tableMatches {
    if !sqlKeywords[strings.ToLower(match)] {
        tables = append(tables, match)
    }
    }
    
    return tables
}

// extractSqlColumns extracts column names from a SELECT statement
func extractSqlColumns(stmt string) []string {
    var columns []string
    
    lowerStmt := strings.ToLower(stmt)
    selectPos := strings.Index(lowerStmt, "select")
    fromPos := strings.Index(lowerStmt, "from")
    
    if selectPos == -1 || fromPos == -1 || selectPos >= fromPos {
    return columns
    }
    
    // Extract text between SELECT and FROM
    columnsText := stmt[selectPos+6:fromPos]
    
    // Handle SELECT *
    if strings.TrimSpace(columnsText) == "*" {
    columns = append(columns, "*")
    return columns
    }
    
    // Split by commas
    columnParts := strings.Split(columnsText, ",")
    
    for _, part := range columnParts {
    part = strings.TrimSpace(part)
    
    // Handle column aliases
    if strings.Contains(part, " as ") {
        parts := strings.Split(strings.ToLower(part), " as ")
        if len(parts) >= 2 {
	part = strings.TrimSpace(parts[1])
        }
    }
    
    // Extract the column name
    colRegex := regexp.MustCompile(`[a-zA-Z0-9_]+`)
    colMatches := colRegex.FindAllString(part, -1)
    
    if len(colMatches) > 0 {
        columns = append(columns, colMatches[len(colMatches)-1])
    }
    }
    
    return columns
}

// extractSqlInsertColumns extracts column names from an INSERT statement
func extractSqlInsertColumns(stmt string) []string {
    var columns []string
    
    // Look for column list in INSERT statement
    colListRegex := regexp.MustCompile(`\(\s*([^)]+)\s*\)\s*VALUES`)
    colListMatch := colListRegex.FindStringSubmatch(stmt)
    
    if len(colListMatch) >= 2 {
    colList := colListMatch[1]
    // Split by commas
    colParts := strings.Split(colList, ",")
    
    for _, part := range colParts {
        part = strings.TrimSpace(part)
        // Extract identifier
        colRegex := regexp.MustCompile(`[a-zA-Z0-9_]+`)
        colMatches := colRegex.FindAllString(part, -1)
        
        if len(colMatches) > 0 {
	columns = append(columns, colMatches[0])
        }
    }
    }
    
    return columns
}

// extractSqlUpdateColumns extracts column names from an UPDATE statement
func extractSqlUpdateColumns(stmt string) []string {
    var columns []string
    
    // Look for SET clause in UPDATE statement
    lowerStmt := strings.ToLower(stmt)
    setPos := strings.Index(lowerStmt, "set ")
    wherePos := strings.Index(lowerStmt, "where ")
    
    if setPos == -1 {
    return columns
    }
    
    // Extract SET clause
    var setClause string
    if wherePos == -1 {
    setClause = stmt[setPos+4:]
    } else {
    setClause = stmt[setPos+4:wherePos]
    }
    
    // Split by commas
    colParts := strings.Split(setClause, ",")
    
    for _, part := range colParts {
    // Look for column name before =
    eqPos := strings.Index(part, "=")
    if eqPos == -1 {
        continue
    }
    
    colName := strings.TrimSpace(part[:eqPos])
    // Extract identifier
    colRegex := regexp.MustCompile(`[a-zA-Z0-9_]+`)
    colMatches := colRegex.FindAllString(colName, -1)
    
    if len(colMatches) > 0 {
        columns = append(columns, colMatches[0])
    }
    }
    
    return columns
}

// extractSqlCreateColumns extracts column definitions from a CREATE TABLE statement
func extractSqlCreateColumns(stmt string) []string {
    var columns []string
    
    // Find column definitions part
    colDefsRegex := regexp.MustCompile(`\(\s*([^)]+)\s*\)`)
    colDefsMatch := colDefsRegex.FindStringSubmatch(stmt)
    
    if len(colDefsMatch) >= 2 {
    colDefs := colDefsMatch[1]
    // Split by commas, but be careful about commas in constraint definitions
    colParts := strings.Split(colDefs, ",")
    
    for _, part := range colParts {
        part = strings.TrimSpace(part)
        
        // Skip if this is a constraint or key definition
        lowerPart := strings.ToLower(part)
        if strings.HasPrefix(lowerPart, "constraint") || 
           strings.HasPrefix(lowerPart, "primary key") || 
           strings.HasPrefix(lowerPart, "foreign key") {
	continue
        }
        
        // Extract column name (should be the first identifier)
        colRegex := regexp.MustCompile(`[a-zA-Z0-9_]+`)
        colMatches := colRegex.FindAllString(part, -1)
        
        if len(colMatches) > 0 {
	columns = append(columns, colMatches[0])
        }
    }
    }
    
    return columns
}

// Utility functions
func countLines(text string) int {
    return 1 + strings.Count(text, "\n")
}

func isWithinString(content string, pos int) bool {
    // Check if the position is within a string literal
    // This is a simplification - a real implementation would track string states
    
    // Count quotes before this position
    singleQuotes := strings.Count(content[:pos], "'") - strings.Count(content[:pos], "\\'")
    doubleQuotes := strings.Count(content[:pos], "\"") - strings.Count(content[:pos], "\\\"")
    
    return singleQuotes%2 == 1 || doubleQuotes%2 == 1
}

func isWithinFunction(content string, pos int) bool {
    // Check if position is inside a function
    // Find last opening and closing braces before this position
    openCount := strings.Count(content[:pos], "{")
    closeCount := strings.Count(content[:pos], "}")
    
    return openCount > closeCount
}

func isWithinClass(content string, pos int) bool {
    // Find last "class" before this position
    lastClassPos := strings.LastIndex(content[:pos], "class ")
    
    if lastClassPos == -1 {
    return false
    }
    
    // Find opening brace after "class"
    openBracePos := strings.Index(content[lastClassPos:pos], "{")
    
    if openBracePos == -1 {
    return false
    }
    
    // Count opening and closing braces between class opening and current position
    classPart := content[lastClassPos+openBracePos+1:pos]
    openCount := strings.Count(classPart, "{")
    closeCount := strings.Count(classPart, "}")
    
    return openCount >= closeCount
}

func isWithinMethod(content string, pos int) bool {
    // Check if position is inside a method
    // Find last "function" before this position
    lastFuncPos := strings.LastIndex(content[:pos], "function ")
    
    if lastFuncPos == -1 {
    return false
    }
    
    // Find opening brace after "function"
    openBracePos := strings.Index(content[lastFuncPos:pos], "{")
    
    if openBracePos == -1 {
    return false
    }
    
    // Count opening and closing braces between function opening and current position
    funcPart := content[lastFuncPos+openBracePos+1:pos]
    openCount := strings.Count(funcPart, "{")
    closeCount := strings.Count(funcPart, "}")
    
    return openCount >= closeCount
}

// findLinkedFunctions finds functions linked to an HTML element
func findLinkedFunctions(element HtmlElement, allFunctions map[string]Function, allClasses map[string]Struct) []string {
    var linkedFunctions []string

    // Check for event handlers in attributes
    for key, value := range element.Attributes {
    if strings.HasPrefix(key, "on") && strings.Contains(value, "(") {
        // Extract function name from event handler like onClick="myFunction()"
        re := regexp.MustCompile(`([a-zA-Z0-9_]+)\(`)
        matches := re.FindStringSubmatch(value)
        if len(matches) > 1 {
	funcName := matches[1]
	if _, exists := allFunctions[funcName]; exists {
	    linkedFunctions = appendIfNotExists(linkedFunctions, funcName)
	}
        }
    }
    }

    // Check for data-* attributes that might reference functions
    if funcName, exists := element.Attributes["data-function"]; exists {
    if _, exists := allFunctions[funcName]; exists {
        linkedFunctions = appendIfNotExists(linkedFunctions, funcName)
    }
    }

    // Check for PHP/JS event handlers
    if handlerName, exists := element.Attributes["onclick"]; exists {
    if _, exists := allFunctions[handlerName]; exists {
        linkedFunctions = appendIfNotExists(linkedFunctions, handlerName)
    }
    }

    // Check for form actions that point to PHP scripts
    if action, exists := element.Attributes["action"]; exists && 
       (strings.HasSuffix(action, ".php") || strings.Contains(action, ".php?")) {
    // Extract the PHP script name
    scriptName := filepath.Base(strings.Split(action, "?")[0])
    scriptName = strings.TrimSuffix(scriptName, ".php")
    
    // Look for functions with matching name
    for funcName := range allFunctions {
        if strings.ToLower(funcName) == strings.ToLower(scriptName) ||
           strings.HasPrefix(strings.ToLower(funcName), strings.ToLower(scriptName)+"_") {
	linkedFunctions = appendIfNotExists(linkedFunctions, funcName)
        }
    }
    }

    // Check for hx-get, hx-post, hx-put, etc. attributes (HTMX)
    for key, value := range element.Attributes {
    if strings.HasPrefix(key, "hx-") && strings.Contains(value, "/") {
        // Extract endpoint path which might map to a handler
        parts := strings.Split(value, "/")
        if len(parts) > 0 {
	lastPart := parts[len(parts)-1]
	// Check if there's a function with a similar name
	for funcName := range allFunctions {
	    // Convert camelCase or snake_case to lowercase for comparison
	    funcNameLower := strings.ToLower(funcName)
	    lastPartLower := strings.ToLower(lastPart)
	    
	    // Remove common suffixes for comparison
	    funcNameLower = strings.TrimSuffix(funcNameLower, "handler")
	    funcNameLower = strings.TrimSuffix(funcNameLower, "endpoint")
	    funcNameLower = strings.TrimSuffix(funcNameLower, "controller")
	    lastPartLower = strings.TrimSuffix(lastPartLower, ".html")
	    lastPartLower = strings.TrimSuffix(lastPartLower, ".php")
	    
	    if strings.Contains(funcNameLower, lastPartLower) || 
	       strings.Contains(lastPartLower, funcNameLower) {
	    linkedFunctions = appendIfNotExists(linkedFunctions, funcName)
	    }
	}
        }
    }
    }

    return linkedFunctions
}

// processPythonFileForPattern extracts pattern information from a Python file
func processPythonFileForPattern(pyFile PythonFileSummary, fileIndex int, pattern *PatternSummary) {
    // Add classes to types
    for _, c := range pyFile.Classes {
        pattern.Types = append(pattern.Types, c.Name)
        pattern.FileMap[c.Name] = append(pattern.FileMap[c.Name], fileIndex)
    }
    
    // Add functions
    for _, f := range pyFile.Functions {
        pattern.Functions = append(pattern.Functions, f.Name)
        pattern.FileMap[f.Name] = append(pattern.FileMap[f.Name], fileIndex)
    }
    
    // Add decorators as special "types"
    for _, decorator := range pyFile.Decorators {
        decoratorName := "@" + decorator
        pattern.Types = append(pattern.Types, decoratorName)
        pattern.FileMap[decoratorName] = append(pattern.FileMap[decoratorName], fileIndex)
    }
}

// convertToPatternFormat converts to the AI-friendly pattern format
func convertToPatternFormat(summary Summary, config Config) PatternSummary {
    patternSummary := PatternSummary{
    Timestamp:   time.Now().Format(time.RFC3339),
    AnalyzedDir: config.Directory,
    FileMap:     make(map[string][]int),
    Files:       make([]string, 0),
    }
    
    // Collect all file paths
    fileIndex := 0
    
    // Go files
    for _, goFile := range summary.GoFiles {
    patternSummary.Files = append(patternSummary.Files, goFile.FilePath)
    processGoFileForPattern(goFile, fileIndex, &patternSummary)
    fileIndex++
    }
    
    // PHP files
    for _, phpFile := range summary.PhpFiles {
    patternSummary.Files = append(patternSummary.Files, phpFile.FilePath)
    processPhpFileForPattern(phpFile, fileIndex, &patternSummary)
    fileIndex++
    }
    
    // Python files
    for _, pyFile := range summary.PythonFiles {
        patternSummary.Files = append(patternSummary.Files, pyFile.FilePath)
        processPythonFileForPattern(pyFile, fileIndex, &patternSummary)
        fileIndex++
    }
    
    // HTML files
    for _, htmlFile := range summary.HtmlFiles {
    patternSummary.Files = append(patternSummary.Files, htmlFile.FilePath)
    processHtmlFileForPattern(htmlFile, fileIndex, &patternSummary)
    fileIndex++
    }
    
    // CSS files
    for _, cssFile := range summary.CssFiles {
    patternSummary.Files = append(patternSummary.Files, cssFile.FilePath)
    processCssFileForPattern(cssFile, fileIndex, &patternSummary)
    fileIndex++
    }
    
    // SQL files
    for _, sqlFile := range summary.SqlFiles {
    patternSummary.Files = append(patternSummary.Files, sqlFile.FilePath)
    processSqlFileForPattern(sqlFile, fileIndex, &patternSummary)
    fileIndex++
    }
    
    // Remove duplicates and sort
    patternSummary.Types = removeDuplicatesAndSort(patternSummary.Types)
    patternSummary.Functions = removeDuplicatesAndSort(patternSummary.Functions)
    patternSummary.CSSSelectors = removeDuplicatesAndSort(patternSummary.CSSSelectors)
    patternSummary.SQLTables = removeDuplicatesAndSort(patternSummary.SQLTables)
    
    // Keep the full details
    patternSummary.Details = summary
    
    return patternSummary
}

// processGoFileForPattern extracts pattern information from a Go file
func processGoFileForPattern(goFile GoFileSummary, fileIndex int, pattern *PatternSummary) {
    // Add structs to types
    for _, s := range goFile.Structs {
    pattern.Types = append(pattern.Types, s.Name)
    pattern.FileMap[s.Name] = append(pattern.FileMap[s.Name], fileIndex)
    }
    
    // Add interfaces to types
    for _, i := range goFile.Interfaces {
    pattern.Types = append(pattern.Types, i.Name)
    pattern.FileMap[i.Name] = append(pattern.FileMap[i.Name], fileIndex)
    }
    
    // Add functions
    for _, f := range goFile.Functions {
    pattern.Functions = append(pattern.Functions, f.Name)
    pattern.FileMap[f.Name] = append(pattern.FileMap[f.Name], fileIndex)
    }
}

// processPhpFileForPattern extracts pattern information from a PHP file
func processPhpFileForPattern(phpFile PhpFileSummary, fileIndex int, pattern *PatternSummary) {
    // Add classes to types
    for _, c := range phpFile.Classes {
    pattern.Types = append(pattern.Types, c.Name)
    pattern.FileMap[c.Name] = append(pattern.FileMap[c.Name], fileIndex)
    }
    
    // Add interfaces to types
    for _, i := range phpFile.Interfaces {
    pattern.Types = append(pattern.Types, i.Name)
    pattern.FileMap[i.Name] = append(pattern.FileMap[i.Name], fileIndex)
    }
    
    // Add functions
    for _, f := range phpFile.Functions {
    pattern.Functions = append(pattern.Functions, f.Name)
    pattern.FileMap[f.Name] = append(pattern.FileMap[f.Name], fileIndex)
    }
}

// processHtmlFileForPattern extracts pattern information from an HTML file
func processHtmlFileForPattern(htmlFile HtmlFileSummary, fileIndex int, pattern *PatternSummary) {
    // Add embedded JS functions
    for _, f := range htmlFile.EmbeddedJS {
    pattern.Functions = append(pattern.Functions, f.Name)
    pattern.FileMap[f.Name] = append(pattern.FileMap[f.Name], fileIndex)
    }
    
    // Add element IDs for reference
    for _, elem := range htmlFile.Elements {
    if elem.ID != "" {
        elemId := "#" + elem.ID
        pattern.FileMap[elemId] = append(pattern.FileMap[elemId], fileIndex)
    }
    }
}

// processCssFileForPattern extracts pattern information from a CSS file
func processCssFileForPattern(cssFile CSSFileSummary, fileIndex int, pattern *PatternSummary) {
    // Add CSS selectors
    for _, rule := range cssFile.Rules {
    pattern.CSSSelectors = append(pattern.CSSSelectors, rule.Selector)
    pattern.FileMap[rule.Selector] = append(pattern.FileMap[rule.Selector], fileIndex)
    }
}

// processSqlFileForPattern extracts pattern information from a SQL file
func processSqlFileForPattern(sqlFile SQLFileSummary, fileIndex int, pattern *PatternSummary) {
    // Add SQL tables
    for _, stmt := range sqlFile.Statements {
    for _, table := range stmt.Tables {
        pattern.SQLTables = append(pattern.SQLTables, table)
        pattern.FileMap[table] = append(pattern.FileMap[table], fileIndex)
    }
    }
}

// filterEmptySlices removes empty slices from the summary
func filterEmptySlices(summary Summary) Summary {
    // Filter Go files
    for i := range summary.GoFiles {
    if len(summary.GoFiles[i].Variables) == 0 {
        summary.GoFiles[i].Variables = nil
    }
    if len(summary.GoFiles[i].Functions) == 0 {
        summary.GoFiles[i].Functions = nil
    }
    if len(summary.GoFiles[i].ControlFlows) == 0 {
        summary.GoFiles[i].ControlFlows = nil
    }
    if len(summary.GoFiles[i].Structs) == 0 {
        summary.GoFiles[i].Structs = nil
    }
    if len(summary.GoFiles[i].Interfaces) == 0 {
        summary.GoFiles[i].Interfaces = nil
    }
    if len(summary.GoFiles[i].Imports) == 0 {
        summary.GoFiles[i].Imports = nil
    }
    }

    // Filter PHP files
    for i := range summary.PhpFiles {
    if len(summary.PhpFiles[i].Variables) == 0 {
        summary.PhpFiles[i].Variables = nil
    }
    if len(summary.PhpFiles[i].Functions) == 0 {
        summary.PhpFiles[i].Functions = nil
    }
    if len(summary.PhpFiles[i].ControlFlows) == 0 {
        summary.PhpFiles[i].ControlFlows = nil
    }
    if len(summary.PhpFiles[i].Classes) == 0 {
        summary.PhpFiles[i].Classes = nil
    }
    if len(summary.PhpFiles[i].Interfaces) == 0 {
        summary.PhpFiles[i].Interfaces = nil
    }
    if len(summary.PhpFiles[i].Imports) == 0 {
        summary.PhpFiles[i].Imports = nil
    }
    }
    
    // Filter Python files
    for i := range summary.PythonFiles {
        if len(summary.PythonFiles[i].Variables) == 0 {
            summary.PythonFiles[i].Variables = nil
        }
        if len(summary.PythonFiles[i].Functions) == 0 {
            summary.PythonFiles[i].Functions = nil
        }
        if len(summary.PythonFiles[i].ControlFlows) == 0 {
            summary.PythonFiles[i].ControlFlows = nil
        }
        if len(summary.PythonFiles[i].Classes) == 0 {
            summary.PythonFiles[i].Classes = nil
        }
        if len(summary.PythonFiles[i].Imports) == 0 {
            summary.PythonFiles[i].Imports = nil
        }
        if len(summary.PythonFiles[i].Decorators) == 0 {
            summary.PythonFiles[i].Decorators = nil
        }
    }
    
    // Filter HTML files
    for i := range summary.HtmlFiles {
    if len(summary.HtmlFiles[i].Elements) == 0 {
        summary.HtmlFiles[i].Elements = nil
    }
    if len(summary.HtmlFiles[i].EmbeddedJS) == 0 {
        summary.HtmlFiles[i].EmbeddedJS = nil
    }
    if len(summary.HtmlFiles[i].EmbeddedCSS) == 0 {
        summary.HtmlFiles[i].EmbeddedCSS = nil
    }
    if len(summary.HtmlFiles[i].Includes) == 0 {
        summary.HtmlFiles[i].Includes = nil
    }
    }
    
    // Filter CSS files
    for i := range summary.CssFiles {
    if len(summary.CssFiles[i].Rules) == 0 {
        summary.CssFiles[i].Rules = nil
    }
    if len(summary.CssFiles[i].Imports) == 0 {
        summary.CssFiles[i].Imports = nil
    }
    }
    
    // Filter SQL files
    for i := range summary.SqlFiles {
    if len(summary.SqlFiles[i].Statements) == 0 {
        summary.SqlFiles[i].Statements = nil
    }
    }

    return summary
}

// removeDuplicatesAndSort removes duplicates from a slice and sorts it
func removeDuplicatesAndSort(slice []string) []string {
    // Use a map to remove duplicates
    uniqueMap := make(map[string]bool)
    for _, item := range slice {
    uniqueMap[item] = true
    }
    
    // Convert back to slice
    result := make([]string, 0, len(uniqueMap))
    for item := range uniqueMap {
    result = append(result, item)
    }
    
    // Sort the slice
    sort.Strings(result)
    
    return result
}

// appendIfNotExists appends a string to a slice if it doesn't already exist
func appendIfNotExists(slice []string, item string) []string {
    for _, s := range slice {
    if s == item {
        return slice
    }
    }
    return append(slice, item)
}