Use is fairly straight forward:

go mod init distiller
go mod tidy
go build distiller.go

Once you have the compiled file you can type the name distiller for a breakdown of command options

An easy way to get started would be this:

distiller -dir=(a directory name here that contains ALL of the files you want to distill down) -format=pattern -output=name.json
    so for instance distiller -dir=dirname -format=pattern -output=name.json
Will give you a single distilled pattern file called name.json which will contained a distilled version of every qualifying file in the dirname directory.

Distiller by Philip Ferreira for AI-Assisted Development
Version: 3.0.2

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
