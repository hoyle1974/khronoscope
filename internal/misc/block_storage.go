package misc

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

type BlockStorage interface {
	SaveData(key string, data []byte) error
	LoadData(key string) ([]byte, error)
	List(path string) ([]string, error)
}

type DiskBlockStorage struct {
	path string
}

func NewDiskBlockStorage(path string) *DiskBlockStorage {
	return &DiskBlockStorage{
		path: path,
	}
}

func (m *DiskBlockStorage) List(path string) ([]string, error) {
	ret := []string{}

	files, err := os.ReadDir(filepath.Join(m.path, path))
	if err != nil {
		return ret, err
	}
	for _, file := range files {
		ret = append(ret, file.Name())
	}
	return ret, err
}

// SaveData compresses the data and saves it to a file with the given key.
func (m *DiskBlockStorage) SaveData(key string, data []byte) error {
	// Ensure the directory exists
	filename := filepath.Join(m.path, key)
	dir := filepath.Dir(filename)
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	// Create the output file
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer file.Close()

	// Create a gzip writer to compress the data before saving
	gzipWriter := gzip.NewWriter(file)
	defer gzipWriter.Close()

	// Write the compressed data to the file
	_, err = gzipWriter.Write(data)
	if err != nil {
		return fmt.Errorf("failed to compress and write data: %v", err)
	}

	return nil
}

// LoadData reads the compressed data from the file and decompresses it.
func (m *DiskBlockStorage) LoadData(key string) ([]byte, error) {
	// Construct the full file path using the key
	filePath := filepath.Join(m.path, key)

	// Open the compressed file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	// Create a gzip reader to decompress the file
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %v", err)
	}
	defer gzipReader.Close()

	// Read the decompressed data
	decompressedData, err := io.ReadAll(gzipReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read decompressed data: %v", err)
	}

	return decompressedData, nil
}

type MemoryBlockStorage struct {
	lock sync.RWMutex
	data map[string][]byte
}

func NewMemoryBlockStorage() *MemoryBlockStorage {
	return &MemoryBlockStorage{
		data: map[string][]byte{},
	}
}

func (m *MemoryBlockStorage) SaveData(key string, data []byte) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.data[key] = data
	return nil
}

func (m *MemoryBlockStorage) LoadData(key string) ([]byte, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	if data, ok := m.data[key]; ok {
		return data, nil
	}
	return nil, nil
}
