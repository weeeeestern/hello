
# Go로 grep 구현 정리

grep 은 입력으로 전달된 하나 이상의 내용에서 특정 문자열을 찾고자할 때 사용하는 명령어이다.  
외부에서 파일이름, 옵션, 패턴 등을 받아와야 한다. 받아오는 파일 `config.go`를 `main.go`와 따로 생성한다.

---

## 재귀적으로 디렉토리 탐색

하위 디렉토리까지 모두 검색 대상에 포함한다.

---

## Pipe

pipe : 리눅스나 맥에서 프로그램 간 데이터 스트림을 파이프를 통해 직접 전달함

---

## flag 패키지

flag 패키지 : go 언어의 표준 라이브러리. 명령줄 인자를 쉽게 파싱할 수 있게 해준다.  
명령줄 옵션, 스위치 및 인자를 정의하고, 사용자의 명령줄 입력을 통해 프로그램 행동을 동적으로 관리할 수 있다.

- `flag.BoolVar` : 특정 플래그 정의. 특정 bool 플래그가 명령줄에 존재하는지 감지

### 예시

- `-R` 플래그 : 이 플래그가 활성화되면, 프로그램은 지정된 디렉토리뿐만 아니라 그 하위 모든 디렉토리 트리를 탐색하게 된다.
- `-c` 플래그 : 특정 패턴과 일치하는 줄의 수를 카운트하는 기능을 활성화

---

## if문 내 변수 선언

Go에서는 if문 내에서 변수를 선언하고, 그 변수를 바로 조건문에 사용할 수 있다.

```go
if fi, _ := os.Stdin.Stat(); !dereferenceRecursive && (fi.Mode()&os.ModeCharDevice) == 0 {
````

* `fi` 에는 Stdin의 상태정보를 담는다.
* `_` 는 에러값을 무시하겠다는 의미이다.
* `os.ModeCharDevice` 연산 결과가 0이면 표준 입력이 char가 아니라는 뜻. 즉, 파이프 등을 통해 데이터 스트림이 들어오고 있다는 의미이다.

---

## 임시 파일 생성 및 에러 처리

```go
tmpFile, err := os.CreateTemp("", "gorep")
if err != nil {
    return config, errors.New(cannotCreatTemp)
}
```

* `os.CreateTemp` : 임시파일을 생성한다.
* 첫 번째 인자는 파일이 생성될 디렉토리를 지정하는데, 빈 문자열은 시스템의 기본 임시 디렉토리에 파일이 추가된다.

```go
defer tmpFile.Close()
```

파일을 열면 닫아줘야 한다.
`defer` 함수의 경우 return 문 직전에 실행된다.

---

## 객체 반환 (configuration)

성공적으로 반환 못 하면 `nil`을 반환한다. 에러 상황.

```go
return configuration{
    dereferenceRecursive: dereferenceRecursive,
    count:                count,
    pattern:              tail[0],
    fileName:             []string{tmpFile.Name()},
    fromPipe:             true,
}, nil
```

```go
else if len(tail) < 2 {
    return config, errors.New(invalidArgument)
}
```

```go
else {
    return configuration{
        dereferenceRecursive: dereferenceRecursive,
        count:                count,
        pattern:              tail[0],
        fileName:             tail[1:],
        fromPipe:             false,
    }, nil
}
```

---

## 성능 최적화 : 값 vs 포인터 반환

Go에서는 변수들이 기본적으로 스택에 저장된다.
Java는 heap에 저장된다고 한다.

* 스택 : 메모리 할당과 해제가 빠르며, 함수 호출 시 생성되면서 함수가 끝나면 제거된다.
* 힙 : 동적으로 할당되는 메모리 영역으로, 포인터를 통해 접근된다. 가비지 컬렉터가 이 메모리 관리를 담당한다.

따라서 Go에서는 일반적으로 값 형태로 데이터를 반환하는 것이 권장된다.
값으로 반환하면 데이터의 복사본이 생성되어 반환된다.

해당 함수 밖에서 configuration 객체에 구현된 구조체나 슬라이스를 계속 사용하고 싶은 경우에는 포인터를 return 하겠지. 그러면 느려진다.

---

# Search 함수

```go
file, err := os.Open(filename)
defer file.Close()
```

```go
fileScanner := bufio.NewScanner(file)
```

```go
for fileScanner.Scan() {
    line := fileScanner.Text()
    if index := strings.Index(line, pattern); index > -1 {
        if printPrefix {
            fmt.Printf(searchResultTemplate, file.Name(), lineNumber, line)
        } else {
            fmt.Println(line)
        }
    }
    lineNumber++
}
```

* `strings.Index()` 함수를 사용해 line에 pattern이 포함되어 있는지 검사한다.
* pattern이 처음 나타나는 인덱스를 반환한다.
* 찾지 못하면 -1 반환.
* `printPrefix`는 pattern을 찾았을 때 결과 출력 형식을 제어한다.

---

# config.go

```go
package main

type configuration struct {
	dereferenceRecursive bool // -R 여부
	count                bool // -c 여부
	pattern              string // 받은 pattern 값
	fileName             []string // 파일 이름. 복수의 파일 가능
	fromPipe             bool // pipe를 통해 값을 전달받는지 여부
}

const invalidArgument = "invalid argument count"
const cannotCreatTemp = "cannot create temp file"
const cannotReadFromPipe = "cannot read from pipe"
const cannotWriteTemp = "cannot write to temp file"

// 외부 입력 값들을 받아오는 함수입니다.
func setup() (configuration, error) {
	var config configuration
	var dereferenceRecursive, count bool

	flag.BoolVar(&dereferenceRecursive, "R", false, "dereference recursive")
	flag.BoolVar(&count, "c", false, "count")
	flag.Parse()

	tail := flag.Args()

	if fi, _ := os.Stdin.Stat(); !dereferenceRecursive && (fi.Mode()&os.ModeCharDevice) == 0 {

		tmpFile, err := os.CreateTemp("", "gorep")
		if err != nil {
			return config, errors.New(cannotCreatTemp)
		}

		line, err := io.ReadAll(os.Stdin)
		if err != nil {
			return config, errors.New(cannotReadFromPipe)
		}

		err = os.WriteFile(tmpFile.Name(), line, 0644)
		if err != nil {
			return config, errors.New(cannotWriteTemp)
		}

		defer tmpFile.Close()

		return configuration{
			dereferenceRecursive: dereferenceRecursive,
			count:                count,
			pattern:              tail[0],
			fileName:             []string{tmpFile.Name()},
			fromPipe:             true,
		}, nil

	} else if len(tail) < 2 {

		return config, errors.New(invalidArgument)

	} else {

		return configuration{
			dereferenceRecursive: dereferenceRecursive,
			count:                count,
			pattern:              tail[0],
			fileName:             tail[1:],
			fromPipe:             false,
		}, nil
	}
}
```

---

# main.go

```go
package main

func main() {

	config, err := setup()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(2)
	}

	if config.dereferenceRecursive {
		paths := make([]string, 0)
		err := filepath.Walk(config.fileName[0],
			func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !info.IsDir() {
					paths = append(paths, path)
				}
				return nil
			})
		if err != nil {
			fmt.Errorf(err.Error())
			os.Exit(2)
		}
		config.fileName = paths
	}

	if printPrefix := len(config.fileName) > 1; config.count {
		for _, fileName := range config.fileName {
			searchCount(fileName, config.pattern, printPrefix)
		}
	} else {
		for _, fileName := range config.fileName {
			search(fileName, config.pattern, printPrefix)
		}
	}
}
```

---

# search.go

```go
package main

const cannotReadFile = "unable to read file: %v"
const errorWhileReadFile = "Error while reading file: %s"
const searchResultTemplate = "%s:%d|%s\n"
const countResultTemplate = "%s:%d\n"

func search(filename, pattern string, printPrefix bool) {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatalf(cannotReadFile, err)
	}

	defer file.Close()
	fileScanner := bufio.NewScanner(file)

	lineNumber := 0
	for fileScanner.Scan() {
		line := fileScanner.Text()
		if index := strings.Index(line, pattern); index > -1 {
			if printPrefix {
				fmt.Printf(searchResultTemplate, file.Name(), lineNumber, line)
			} else {
				fmt.Println(line)
			}
		}
		lineNumber++
	}
	if err := fileScanner.Err(); err != nil {
		log.Fatalf(errorWhileReadFile, err)
	}
}

func searchCount(filename, pattern string, printPrefix bool) {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatalf(cannotReadFile, err)
	}

	defer file.Close()
	fileScanner := bufio.NewScanner(file)

	count := 0
	for fileScanner.Scan() {
		line := fileScanner.Text()
		count += strings.Count(line, pattern)
	}

	if err := fileScanner.Err(); err != nil {
		log.Fatalf(errorWhileReadFile, err)
	}

	if count != 0 {
		if printPrefix {
			fmt.Printf(countResultTemplate, file.Name(), count)
		} else {
			fmt.Println(count)
		}
	}
}
```

---

## ref)

[https://namkyu1999.github.io/posts/from-scratch/230211_linux_command_from_scratch_1/](https://namkyu1999.github.io/posts/from-scratch/230211_linux_command_from_scratch_1/)


