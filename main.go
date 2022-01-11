package main

import (
	"context"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/1pkg/gopium/gopium"
	"github.com/1pkg/gopium/runners"
	"github.com/spf13/cobra"
)

var (
	// cli command iteself
	cli *cobra.Command
	// target platform vars
	tcompiler string
	tarch     string
	tcpulines []int
	// package parser vars
	ppath   string
	pbenvs  []string
	pbflags []string
	// gopium walker vars
	wregex   string
	wdeep    bool
	wbackref bool
	// gopium printer vars
	pindent   int
	ptabwidth int
	pusespace bool
	pusegofmt bool
	// gopium global vars
	timeout     int
	packageName string
	batchSize   int
)

const ConfigKey = "x-config"

type Config struct {
	packageName string
	ppath       string
}

func getPackageName(file string) (string, error) {
	fset := token.NewFileSet()

	// parse the go soure file, but only the package clause
	astFile, err := parser.ParseFile(fset, file, nil, parser.PackageClauseOnly)
	if err != nil {
		return "", fmt.Errorf("parse file: %w", err)
	}

	if astFile.Name == nil {
		return "", fmt.Errorf("no package name found")
	}

	return astFile.Name.Name, nil
}

func run(ctx context.Context, args []string) error {
	ourConfig := ctx.Value(ConfigKey).(Config)
	packageName = ourConfig.packageName
	ppath = ourConfig.ppath
	fmt.Println("processing >>", packageName, ppath, args)
	// create cli app instance
	cli, err := runners.NewCli(
		// target platform vars
		tcompiler,
		tarch,
		tcpulines,
		// package parser vars
		packageName, // package name
		ppath,
		pbenvs,
		pbflags,
		// gopium walker vars
		"ast_go", // single walker
		wregex,
		wdeep,
		wbackref,
		args, // strategies slice
		// gopium printer vars
		pindent,
		ptabwidth,
		pusespace,
		pusegofmt,
		// gopium global vars
		timeout,
	)
	if err != nil {
		return err
	}
	// execute app
	return cli.Run(ctx)
}

// init cli command runner
// and global context
func init() {
	// set root cli command app
	cli = &cobra.Command{
		Use:     "gopium -flag_0 -flag_n walker package strategy_1 strategy_2 strategy_3 ...",
		Short:   gopium.STAMP,
		Version: gopium.VERSION,
		Example: "gopium -r ^A go_std 1pkg/gopium filter_pads memory_pack separate_padding_cpu_l1_top separate_padding_cpu_l1_bottom",
		Args:    cobra.MinimumNArgs(1),
	}
	// set target_compiler flag
	cli.Flags().StringVarP(
		&tcompiler,
		"target_compiler",
		"c",
		"gc",
		"Gopium target platform compiler, possible values are: gc or gccgo.",
	)
	// set target_architecture flag
	cli.Flags().StringVarP(
		&tarch,
		"target_architecture",
		"a",
		"amd64",
		"Gopium target platform architecture, possible values are: 386, arm, arm64, amd64, mips, etc.",
	)
	// set target_cpu_cache_lines_sizes flag
	cli.Flags().IntSliceVarP(
		&tcpulines,
		"target_cpu_cache_lines_sizes",
		"l",
		[]int{64, 64, 64},
		`
Gopium target platform CPU cache line sizes in bytes, cache line size is set one by one l1,l2,l3,...
For now only 3 lines of cache are supported by strategies.
		`,
	)
	// set package_path flag
	cli.Flags().StringVarP(
		&ppath,
		"package_path",
		"p",
		filepath.Join("src", "{{package}}"),
		`
Gopium go package path, either relative or absolute path to root of the package is expected.
To obtain full path from relative, package path is concatenated with current GOPATH env var.
Template {{package}} part is replaced with package name.
		`,
	)
	// set package_build_envs flag
	cli.Flags().StringSliceVarP(
		&pbenvs,
		"package_build_envs",
		"e",
		[]string{},
		"Gopium go package build envs, additional list of building envs is expected.",
	)
	// set package_build_flags flag
	cli.Flags().StringSliceVarP(
		&pbflags,
		"package_build_flags",
		"f",
		[]string{},
		"Gopium go package build flags, additional list of building flags is expected.",
	)
	// set walker_regexp flag
	cli.Flags().StringVarP(
		&wregex,
		"walker_regexp",
		"r",
		".*",
		`
Gopium walker regexp, regexp that defines which structures are subjects for visiting.
Visiting is done only if structure name matches the regexp.
		`,
	)
	// set walker_deep flag
	cli.Flags().BoolVarP(
		&wdeep,
		"walker_deep",
		"d",
		true,
		`
Gopium walker deep flag, flag that defines type of nested scopes visiting.
By default it visits all nested scopes.
		`,
	)
	// set walker_backref flag
	cli.Flags().BoolVarP(
		&wbackref,
		"walker_backref",
		"b",
		true,
		`
Gopium walker backref flag, flag that defines type of names referencing.
By default any previous visited types have affect on future relevant visits.
		`,
	)
	// set printer_indent flag
	cli.Flags().IntVarP(
		&pindent,
		"printer_indent",
		"i",
		0,
		"Gopium printer width of tab, defines the least code indent.",
	)
	// set printer_tab_width flag
	cli.Flags().IntVarP(
		&ptabwidth,
		"printer_tab_width",
		"w",
		8,
		"Gopium printer width of tab, defines width of tab in spaces for printer.",
	)
	// set printer_use_space flag
	cli.Flags().BoolVarP(
		&pusespace,
		"printer_use_space",
		"s",
		false,
		"Gopium printer use space flag, flag that defines if all formatting should be done by spaces.",
	)
	// set printer_use_gofmt flag
	cli.Flags().BoolVarP(
		&pusegofmt,
		"printer_use_gofmt",
		"g",
		true,
		`
Gopium printer use gofmt flag, flag that defines if canonical gofmt tool should be used for formatting.
By default it is used and overrides other printer formatting parameters.
`,
	)
	// set timeout flag
	cli.Flags().IntVarP(
		&timeout,
		"timeout",
		"t",
		0,
		"Gopium global timeout of cli command in seconds, considered only if value greater than 0.",
	)
	cli.Flags().IntVarP(
		&batchSize,
		"batch_size",
		"n",
		1,
		"Number of file to scan in parallel",
	)
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	// prepare context with signals cancelation
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	cli.Execute()
	cli.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		return run(ctx, args)
	}

	firstFile := make(map[string]string)
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Println(err)
			return err
		}
		if strings.HasSuffix(path, ".go") {
			absPath, e := filepath.Abs(path)
			if e != nil {
				fmt.Println(e)
				return nil
			}
			paths := strings.Split(absPath, "/")
			if len(paths) < 2 {
				return nil
			}
			dir := strings.Join(paths[:len(paths)-1], "/")
			if _, ok := firstFile[dir]; ok {
				return nil
			}
			firstFile[dir] = path
		}

		return nil
	})
	if err != nil {
		panic(err)
	}
	var wg sync.WaitGroup

	count := 0
	for dir, filename := range firstFile {
		dir := dir
		filename := filename
		wg.Add(1)
		go func() {
			defer wg.Done()
			pname, err := getPackageName(filename)
			if err != nil {
				fmt.Println(err)
				return
			}
			select {
			case <-ctx.Done():
				return
			default:
				newCtx := context.WithValue(ctx, ConfigKey, Config{
					packageName: pname,
					ppath:       dir,
				})
				if err := cli.ExecuteContext(newCtx); err != nil {
					fmt.Println(err)
					return
				}
			}

		}()
		count += 1
		if count == batchSize {
			wg.Wait()
			count = 0
		}
	}
}
