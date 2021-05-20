package code

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"go/types"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

type SourceFileInfo struct {
	Pkg *Package // to remove one field in Identifier. Also good to external source line generation.

	// Filename only.
	BareFilename          string
	BareGeneratedFilename string

	// One and only one of the following two is not blank.
	//NonGoFile      string
	//OriginalGoFile string

	// The full path of a (Go or others) source file.
	// It might be blank for some cgo generated files.
	OriginalFile string

	// The followings are blank for most files.
	GeneratedFile string
	//GoFileContentOffset int32
	//GoFileLineOffset    int32

	// Non-nil for Go files.
	// If an original Go file has a corresponding generated file,
	// then the ast file is for that generated file.
	AstFile *ast.File

	// ...
	Content []byte
}

func (info *SourceFileInfo) AstBareFileName() string {
	if info.BareGeneratedFilename != "" {
		return info.BareGeneratedFilename
	}
	return info.BareFilename
}

func (d *CodeAnalyzer) collectSourceFiles() {
	//log.Println("=================== collectSourceFiles")

	//d.sourceFile2PackageTable = make(map[string]SourceFile, len(d.packageList)*5)
	//d.sourceFile2PackageTable = make(map[string]*Package, len(d.packageList)*5)
	//d.generatedFile2OriginalFileTable = make(map[string]string, 128)
	//d.sourceFileLineOffsetTable = make(map[string]int32, 256)
	for _, pkg := range d.packageList {
		//log.Println("====== ", pkg.Path())
		//if pkg.Path() == "unsafe" {
		//	//log.Println("///============== ", pkg.PPkg.GoFiles)
		//	//ast.Print(pkg.PPkg.Fset, pkg.PPkg.Syntax[0])
		//
		//	// For unsafe package, pkg.PPkg.CompiledGoFiles is blank.
		//	// ToDo: fill it in fillUnsafePackage? (Done)
		//
		//	path := pkg.PPkg.GoFiles[0]
		//
		//	d.sourceFile2PackageTable[path] = SourceFile{
		//		Path:    path,
		//		Pkg:     pkg,
		//		AstFile: pkg.PPkg.Syntax[0],
		//	}
		//
		//	continue
		//}

		if len(pkg.PPkg.CompiledGoFiles) != len(pkg.PPkg.Syntax) {
			panic(fmt.Sprintf("!!! len(pkg.PPkg.CompiledGoFiles) != len(pkg.PPkg.Syntax), %d:%d, %s", len(pkg.PPkg.CompiledGoFiles), len(pkg.PPkg.Syntax), pkg.Path()))
		}

		//for range pkg.PPkg.OtherFiles {
		//	//d.sourceFile2PackageTable[path] = pkg
		//	d.stats.FilesWithoutGenerateds++
		//}
		d.stats.FilesWithoutGenerateds += int32(len(pkg.PPkg.OtherFiles))

		//for range pkg.PPkg.CompiledGoFiles {
		//	//d.sourceFile2PackageTable[path] = pkg
		//}

		for _, path := range pkg.PPkg.GoFiles {
			//	//if _, ok := d.sourceFile2PackageTable[path]; !ok {
			//	//	//log.Println("! in GoFiles but not CompiledGoFiles:", path)
			//	//	d.sourceFile2PackageTable[path] = pkg
			//	//}
			//	d.stats.FilesWithoutGenerateds++
			if pkg.Directory == "" {
				pkg.Directory = filepath.Dir(path)
				break
			}
		}
		d.stats.FilesWithoutGenerateds += int32(len(pkg.PPkg.GoFiles))

		//d.collectSourceFileInfos(pkg)
		if pkg.SourceFiles != nil {
			return
		}
		func() {
			pkg.SourceFiles = make([]SourceFileInfo, 0, len(pkg.PPkg.CompiledGoFiles))

			for i, compiledFile := range pkg.PPkg.CompiledGoFiles {
				if strings.HasSuffix(compiledFile, ".go") {
					// ToDo: verify compiledFile must be also in  pkg.PPkg.GoFiles
					pkg.SourceFiles = append(pkg.SourceFiles,
						SourceFileInfo{
							Pkg:           pkg,
							BareFilename:  filepath.Base(compiledFile),
							OriginalFile:  compiledFile,
							GeneratedFile: "", //compiledFile,
							AstFile:       pkg.PPkg.Syntax[i],
						},
					)
					continue
				}

				//info := generatedFileInfo(pkg, compiledFile, pkg.PPkg.Syntax[i])
				//
				//if info.OriginalFile != "" && info.GeneratedFile != info.OriginalFile {
				//	d.generatedFile2OriginalFileTable[info.GeneratedFile] = info.OriginalFile
				//}
				//
				//info.AstFile = pkg.PPkg.Syntax[i]

				info := generatedFileInfo(pkg, compiledFile, pkg.PPkg.Syntax[i])
				pkg.SourceFiles = append(pkg.SourceFiles, info)

				//if info.GoFileLineOffset != 0 {
				//	d.sourceFileLineOffsetTable[info.OriginalGoFile] = info.GoFileLineOffset
				//}
			}

			for _, path := range pkg.PPkg.OtherFiles {
				pkg.SourceFiles = append(pkg.SourceFiles,
					SourceFileInfo{
						Pkg:          pkg,
						BareFilename: filepath.Base(path),
						OriginalFile: path,
					},
				)
			}
		}()

		////d.stats.Files += int32(len(pkg.SourceFiles))
		//d.stat_OnNewPackage(d.IsStandardPackage(pkg), len(pkg.SourceFiles), len(pkg.Deps), pkg.Path())
		d.stat_OnNewPackage(d.IsStandardPackage(pkg), len(pkg.PPkg.CompiledGoFiles), len(pkg.Deps), pkg.Path())
	}
}

//==================================

// ToDo: find a better way to detect generated files.
var cgoGenIdent = []byte(`// Code generated by cmd/cgo; DO NOT EDIT.`)
var reposIdent = []byte(`//line `)

// https://github.com/golang/go/issues/24183
// https://github.com/golang/go/issues/26207
// https://github.com/golang/go/issues/36072
// The function is not robust enough to handle all kinds of special cases.
// 1. It doesn't consider /*line file:m:n*/ form.
// 2. It doesn't handle multiple "//line ..." occurrences.
// 3. It should ignore the general content enclosed in other comments.
//    /*
//    //line file:m:n
//    */
// Maybe it is best to check the comments nodes in the already provided ast.File.
func generatedFileInfo(pkg *Package, filename string, astFile *ast.File) SourceFileInfo {
	var goFilename string

	fileContent, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Printf("generatedFileInfo: ReadFile(%s) error: %s", filename, err)
		fileContent = []byte{} // ToDo: might be not needed. To read ioutil.ReadFile docs.
		goto Done
	}

	// ToDo: the current implement strongly depends on the file genenration implementation.
	//       Here it is assumed that the "// Code generated by cmd/cgo; DO NOT EDIT." is the 4th line.
	if !bytes.HasPrefix(fileContent, cgoGenIdent) {
		goto Done
	}

	// Scan at most 8 lines.
	for lineNumber, data := 1, fileContent; lineNumber <= 8 && len(data) > 0; lineNumber++ {
		i := bytes.IndexByte(data, '\n')
		k := i
		if k < 0 {
			k = len(data)
		}
		if k > 0 && data[k-1] == '\r' {
			k--
		}
		for bytes.HasPrefix(data[:k], reposIdent) {
			line := bytes.TrimSpace(data[len(reposIdent):k])
			indexB := bytes.LastIndexByte(line, ':')
			if indexB < 0 {
				break
			}

			// Assume the colume offset is 1.
			if indexA := bytes.LastIndexByte(line[:indexB], ':'); indexA >= 0 {
				goFilename = string(line[:indexA])
			} else {
				goFilename = string(line[:indexB])
			}
			goto Done
		}
		if i >= 0 {
			data = data[i+1:]
		}
	}

	log.Printf("??? generatedFileInfo: file (%s) looks like cgo generated but the original file not found", filename)

Done:
	var barFilename string
	if goFilename != "" {
		barFilename = filepath.Base(goFilename)
	} else {
		barFilename = filepath.Base(filename)
	}

	return SourceFileInfo{
		Pkg:           pkg,
		BareFilename:  barFilename,
		OriginalFile:  goFilename,
		GeneratedFile: filename,
		AstFile:       astFile,
		Content:       fileContent,
	}
}

func (d *CodeAnalyzer) collectObjectReferences() {
	for _, pkg := range d.packageList {
		for i := range pkg.SourceFiles {
			info := &pkg.SourceFiles[i]
			// This if-block is still needed for std packages.
			// For other packages, this field has been set in confirmPackageModules.
			if pkg.Directory == "" && info.OriginalFile != "" {
				pkg.Directory = filepath.Dir(info.OriginalFile)
			}
			//log.Println("===", info.OriginalGoFile)
			//log.Println("   ", info.GeneratedFile, info.GoFileContentOffset)
			if info.AstFile == nil {
				continue
			}
			d.collectIdentiferFromFile(pkg, info)
		}
	}
}

func (d *CodeAnalyzer) collectIdentiferFromFile(pkg *Package, fileInfo *SourceFileInfo) {
	ast.Inspect(fileInfo.AstFile, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.Ident:
			obj := pkg.PPkg.TypesInfo.ObjectOf(n)
			if obj != nil {
				d.regObjectReference(obj, fileInfo, n)
				if v, ok := obj.(*types.Var); ok && v.Embedded() {
					obj = pkg.PPkg.TypesInfo.Uses[n]
					if obj != nil {
						d.regObjectReference(obj, fileInfo, n)
					}
				}
			}
			// ToDo: more implicit cases?
		}
		return true
	})
}

// ToDo: can we get the content from the collected AST files?
//       Need to hack the std packages?
func (d *CodeAnalyzer) cacheSourceFiles() {
	n := runtime.GOMAXPROCS(-1)
	sem := make(chan struct{}, n)
	var wg sync.WaitGroup
	defer wg.Wait()

	for _, pkg := range d.packageList {
		//isUnsafe := pkg.Path() == "unsafe"
		for i := range pkg.SourceFiles {
			info := &pkg.SourceFiles[i]
			if info.Content != nil {
				continue
			}

			wg.Add(1)

			filePath := info.OriginalFile
			if info.GeneratedFile != "" {
				filePath = info.GeneratedFile
			}

			sem <- struct{}{}
			go func() { //isUnsafeDotGo bool) {
				defer func() {
					<-sem
					wg.Done()
				}()

				var content []byte
				//if isUnsafeDotGo {
				//content = unsafe_go
				//} else {
				var err error
				content, err = ioutil.ReadFile(filePath)
				if err != nil {
					log.Printf("ReadFile (%s) error: %s", filePath, err)
					return
				}
				//}
				info.Content = content
				//log.Printf("ReadFile (%s) done", filePath)
			}() //isUnsafe && filePath == "unsafe.go")
		}
	}
}

func (d *CodeAnalyzer) collectCodeExamples() {
	collectExampleFiles := func(pkg *Package) []string {
		filenames := make([]string, 0, 8)
		first := true
		if err := filepath.Walk(pkg.Directory, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				if first {
					first = false
					return nil
				}
				return filepath.SkipDir
			}
			name := info.Name()
			if strings.HasPrefix(name, "example_") && strings.HasSuffix(name, "_test.go") {
				filenames = append(filenames, name)
			}
			return nil
		}); err != nil {
			log.Printf("walk package %s dir %s error: %s", pkg.Path(), pkg.Directory, err)
		}
		return filenames
	}

	d.exampleFileSet = token.NewFileSet()
	for _, pkg := range d.packageList {
		if pkg.ExampleFiles != nil {
			continue
		}
		filenames := collectExampleFiles(pkg)
		pkg.ExampleFiles = make([]*ast.File, 0, len(filenames))
		for _, f := range filenames {
			f := filepath.Join(pkg.Directory, f)
			astFile, err := parser.ParseFile(d.exampleFileSet, f, nil, parser.ParseComments)
			if err != nil {
				fmt.Printf("parse file %s error: %s", f, err)
				continue
			}
			pkg.ExampleFiles = append(pkg.ExampleFiles, astFile)
		}
		pkg.Examples = doc.Examples(pkg.ExampleFiles...)
	}
}
