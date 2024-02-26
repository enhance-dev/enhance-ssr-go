package main

import (
	"context"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"github.com/extism/go-sdk"
)

func main() {
	http.HandleFunc("/", handleRequest)
  fmt.Println("Server starting on http://localhost:8080 ...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("Error starting server: %s\n", err)
	}
}


// Marshal formats the JSON output with indentation and without escaping HTML characters.
// It mimics json.MarshalIndent but without HTML escaping.
func Marshal(i interface{}) ([]byte, error) {
    buffer := &bytes.Buffer{}
    encoder := json.NewEncoder(buffer)
    encoder.SetEscapeHTML(false)
    encoder.SetIndent("", "    ") 
    err := encoder.Encode(i)
    if err != nil {
        return nil, err 
    }
    // Trim the trailing newline added by Encode
    return bytes.TrimRight(buffer.Bytes(), "\n"), nil
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	markup := "<my-header>Hello World</my-header>"
	elementPath := "./elements"
	elements := readElements(elementPath)
	initialState := make(map[string]interface{})
	data := map[string]interface{}{
		"markup":       markup,
		"elements":     elements,
		"initialState": initialState,
	}
	payload, err := Marshal(data)
	if err != nil {
		http.Error(w, "Failed to create payload", http.StatusInternalServerError)
		return
	}

	rendered, err := render(payload)
	if err != nil {
		http.Error(w, "Failed to render document", http.StatusInternalServerError)
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(rendered, &result); err != nil {
		http.Error(w, "Failed to parse rendered output", http.StatusInternalServerError)
		return
	}

	document, ok := result["document"].(string)
	if !ok {
		http.Error(w, "Rendered document is not a string", http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, document)
}

func readElements(directory string) map[string]string {
	elements := make(map[string]string)
	files, err := os.ReadDir(directory)
	if err != nil {
		fmt.Printf("Error reading directory: %s\n", err)
		return elements
	}

	for _, file := range files {
		if !file.IsDir() {
			filePath := filepath.Join(directory, file.Name())
			content, err := os.ReadFile(filePath)
			if err != nil {
				fmt.Printf("Error reading file %s: %s\n", file.Name(), err)
				continue
			}
			key := filepath.Base(filePath)
			ext := filepath.Ext(key)
			keyWithoutExt := key[:len(key)-len(ext)]
			elements[keyWithoutExt] = string(content)
		}
	}
	return elements
}

func render(payload []byte) ([]byte, error) {
	manifest := extism.Manifest{
		Wasm: []extism.Wasm{
      extism.WasmFile{
				Path: "./enhance-ssr.wasm",
			},
		},
	}

	ctx := context.Background()
	config := extism.PluginConfig{
    EnableWasi: true,
  }
	plugin, err := extism.NewPlugin(ctx, manifest, config, []extism.HostFunction{})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize plugin: %v", err)
	}

	exit, out, err := plugin.Call("ssr", payload)
	if err != nil {
		return nil, fmt.Errorf("plugin call failed: %v, exit code: %d", err, exit)
	}

	return out, nil
}
