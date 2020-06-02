package parseprotobuf

// Parses a protobuf file and creates a client go file
import (
	"bufio"
	"fmt"
	log "github.com/cihub/seelog"
	"io"
	"os"
	"strings"
)

//Prints the filtered contents of the protobuf file
func PrintHint(phArr []ProtoHint) {
	log.Info("Protobuf Hint: ")
	for _, ph := range phArr {
		if ph.Root {
			for name, srct := range ph.Contents {
				if srct.Root {
					log.Info(name)
					PrintContents(srct, phArr, "\t")
				}
			}
		}
	}
}

//Prints (and returns) a json representation of the request protobuf object
func PrintJsonExample(phArr []ProtoHint) (string, error) {
	log.Info("Example JSON")
	jsonContent := ""
	for _, ph := range phArr {
		if ph.Root {
			for name, srct := range ph.Contents {
				if srct.Root && name == "Request" {
					log.Info(name)
					jsonContent = PrintJsonContent(srct, phArr)
				}
			}
		}
	}
	log.Info(jsonContent)
	return jsonContent, nil
}

//recursively constructs json from the protoStruct
func PrintJsonContent(ps ProtoStruct, phArr []ProtoHint) string {
	obj := "{"
	i := 0
	for _, val := range ps.Values {
		trail := "," //Write comma's between objects
		if len(ps.Values)-1 == i {
			trail = "" //No comma on the last object
		}
		i++
		arrOpenStr := ""
		arrCloseStr := ""
		if val.Req == "repeated" { //write the open and close braces for arrays
			arrOpenStr = "["
			arrCloseStr = "]"
		}
		switch val.Type {
		case "string":
			obj += fmt.Sprintf(`"%s": %s"%s"%s%s `, val.Name, arrOpenStr, "", arrCloseStr, trail)
		case "bool":
			obj += fmt.Sprintf(`"%s": %s%v%s%s `, val.Name, arrOpenStr, "", arrCloseStr, trail)
		case "int32", "int64", "float", "double", "uint64", "bytes", "sfixed64", "sfixed32", "fixed64", "fixed32", "sint64", "sint32", "uint32":
			obj += fmt.Sprintf(`"%s": %s%v%s%s `, val.Name, arrOpenStr, 0, arrCloseStr, trail)
		default:
			obj += fmt.Sprintf(`"%s": %s `, val.Name, arrOpenStr)
			found := false
			for _, ph := range phArr {
				for typeName, typeContents := range ph.Contents {
					typeNameArr := strings.Split(typeName, ".")
					badTypeName := typeNameArr[len(typeNameArr)-1]
					typeNameArr = strings.Split(val.Type, ".")
					badValTypeName := typeNameArr[len(typeNameArr)-1]
					if typeName == val.Type || badTypeName == val.Type || badTypeName == badValTypeName {
						found = true
						obj += PrintJsonContent(typeContents, phArr) + arrCloseStr + trail
						break
					}
				}
				if found {
					break
				}
			}
			if !found {
				for _, ph := range phArr {
					for enumName, enumContents := range ph.Enums {
						enumNameArr := strings.Split(enumName, ".")
						badEnumName := enumNameArr[len(enumNameArr)-1]
						enumNameArr = strings.Split(enumName, ".")
						badValEnumName := enumNameArr[len(enumNameArr)-1]
						if enumName == val.Type || badEnumName == val.Type || badEnumName == badValEnumName {
							found = true
							for _, enumValueName := range enumContents.Values {
								obj += fmt.Sprintf(`"%s"%s%s `, enumValueName, arrCloseStr, trail)
								break
							}
							break
						}
					}
					if found {
						break //This case only exists with naming conflicts... I don't handle these well....
					}
				}
			}
		}
	}
	obj += "}"
	return obj
}

//recursively prints protostruct
func PrintContents(ps ProtoStruct, phArr []ProtoHint, tab string) {
	for _, val := range ps.Values {
		switch val.Type {
		case "string", "int32", "bool", "int64", "float", "double", "uint64", "bytes", "sfixed64", "sfixed32", "fixed64", "fixed32", "sint64", "sint32", "unint32":
			log.Infof("%s%s : %s : %s \n", tab, val.Name, val.Type, val.Req)
			break
		default:
			log.Infof("%s%s : %s : %s \n", tab, val.Name, val.Type, val.Req)
			found := false
			for _, ph := range phArr {
				for typeName, typeContents := range ph.Contents {
					if typeName == val.Type {
						found = true
						tab += "\t"
						PrintContents(typeContents, phArr, tab)
					}
				}
			}
			if !found {
				for _, ph := range phArr {
					for enumName, enumContents := range ph.Enums {
						if enumName == val.Type {
							found = true
							log.Infof("\t%senum: %s\n", tab, enumName)
							for _, enumValueName := range enumContents.Values {
								log.Infof("%s\t\t%s\n", tab, enumValueName)
							}
						}
					}
				}
			}
		}

	}

}

//holds a protobuf file
type ProtoHint struct {
	Name     string
	Contents map[string]ProtoStruct
	Enums    map[string]ProtoEnum
	Root     bool
	Package  string
}

//holds ProtoStruct
type ProtoStruct struct {
	Values map[string]NameType
	Root   bool
}

type ProtoEnum struct {
	Values []string
}

type NameType struct {
	Name string
	Type string
	Req  string
}

//constructs a NameType struct
func NewNameType(name string, aType string) NameType {
	return NameType{Name: name, Type: aType}
}

//Parses a protoc file
func ParseProtobufRaw(r io.ReadCloser, packageName string, root bool, gopath string) []ProtoHint {
	defer r.Close()
	phArr := make([]ProtoHint, 0)
	pb := ProtoHint{Name: packageName, Contents: make(map[string]ProtoStruct), Enums: make(map[string]ProtoEnum), Root: root}

	depth := 0                      //Not really used but potentially useful in future
	typeSlice := make([]string, 0)  //holds the type at this layer
	stateSlice := make([]string, 0) //holds the state we are in (message, enum)

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)

		if line == "" {
			continue
		}

		lineArr := strings.Split(line, " ")
		for i, part := range lineArr {
			if strings.Contains(part, "//") {
				lineArr = lineArr[:i] //Filter out comments
				break
			}
		}
		if len(lineArr) == 0 {
			continue
		}

		if len(lineArr) > 1 {
			if strings.Contains(lineArr[len(lineArr)-1], "}") {
				lineArr[len(lineArr)-1] = strings.Replace(lineArr[len(lineArr)-1], "}", "", -1)
				lineArr = append(lineArr, "}")
			}
			if strings.Contains(lineArr[len(lineArr)-1], "{") {
				lineArr[len(lineArr)-1] = strings.Replace(lineArr[len(lineArr)-1], "{", "", -1)
				lineArr = append(lineArr, "{")
			}
		}

		if lineArr[0] == "package" && len(lineArr) == 2 {
			packageStr := strings.Trim(lineArr[1], `;`)
			pb.Package = packageStr
		}

		if (lineArr[0] == "message" || lineArr[0] == "enum") && len(lineArr) > 2 {
			depth++
			typeSlice = append(typeSlice, lineArr[1])
			stateSlice = append(stateSlice, lineArr[0])
			if lineArr[0] == "message" {
				root := false
				if len(stateSlice) == 1 {
					root = true
				}
				msgName := lineArr[1]
				if !pb.Root {
					msgName = pb.Package + "." + msgName
				}
				typeSlice[len(typeSlice)-1] = msgName
				pb.Contents[msgName] = ProtoStruct{Values: make(map[string]NameType), Root: root}
			}
			if lineArr[0] == "enum" {
				msgName := lineArr[1]
				if !pb.Root {
					msgName = pb.Package + "." + msgName
				}
				typeSlice[len(typeSlice)-1] = msgName
				pb.Enums[msgName] = ProtoEnum{Values: make([]string, 0)}
			}
			continue
		}

		if lineArr[0] == "import" && len(lineArr) > 1 {
			importPath := strings.Trim(lineArr[1], `'";`) //todo: resolve this to correct path
			importPathArr := strings.Split(importPath, "/")
			newPN := strings.Replace(importPathArr[len(importPathArr)-1], ".proto", "", 1)
			importPath = ResolveImportPath(importPath, gopath)
			file, err := GetReader(importPath)
			if err != nil {
				log.Error("could not parse imports:", err)
				continue
			}
			newPH := ParseProtobufRaw(file, newPN, false, gopath)
			phArr = append(phArr, newPH...)
		}
		if lineArr[0] == "}" {
			depth--
			typeSlice = typeSlice[:len(typeSlice)-1]
			stateSlice = stateSlice[:len(stateSlice)-1]
			continue
		}
		if len(stateSlice) > 0 {
			if stateSlice[len(stateSlice)-1] == "message" && len(lineArr) > 3 {
				nt := NameType{Name: lineArr[2], Type: lineArr[1], Req: lineArr[0]}
				pb.Contents[typeSlice[len(typeSlice)-1]].Values[nt.Name] = nt
			}

			if stateSlice[len(stateSlice)-1] == "enum" {
				tmpArr := pb.Enums[typeSlice[len(typeSlice)-1]]
				tmpArr.Values = append(tmpArr.Values, lineArr[0])
				pb.Enums[typeSlice[len(typeSlice)-1]] = tmpArr
			}
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading standard input:", err)
	}

	phArr = append(phArr, pb)
	return phArr
}

//helper function to get a bufio.Reader
func GetReader(filename string) (io.ReadCloser, error) {
	f, err := os.Open(filename)
	if err != nil {
		log.Error("Could not read file:", filename)
		return nil, err
	}
	return f, nil
}

func ResolveImportPath(importPath string, gopath string) string {
	importPath = strings.Trim(importPath, `'";`)
	//importPathArr := strings.Split(importPath, "/")
	prefix := "/src/"
	return gopath + prefix + importPath
}
