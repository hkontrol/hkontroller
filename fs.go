package hkontrol

import (
	"github.com/hkontrol/hkontroller/log"

	"bytes"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type fsStore struct {
	Path string
}

func NewFsStore(dir string) Store {
	// Prepare filesystem directory
	// Ensure that execute permission bit is set on all created dirs
	// Read http://unix.stackexchange.com/questions/21251/why-do-directories-need-the-executable-x-permission-to-be-opened
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		log.Info.Panic(err)
	}

	return &fsStore{dir}
}

func (fs *fsStore) Set(key string, value []byte) error {
	file, err := os.OpenFile(fs.filePathToFile(key), os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}

	defer file.Close()

	_, err = file.Write(value)
	return err
}

func (fs *fsStore) Get(key string) ([]byte, error) {
	file, err := os.OpenFile(fs.filePathToFile(key), os.O_RDONLY, 0666)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	var b bytes.Buffer
	var buffer = make([]byte, 32)
	for {
		n, _ := file.Read(buffer)
		if n > 0 {
			b.Write(buffer[:n])
		} else {
			break
		}
	}

	return b.Bytes(), nil
}

// Delete removes the file for the corresponding key.
func (fs *fsStore) Delete(key string) error {
	return os.Remove(fs.filePathToFile(key))
}

func (fs *fsStore) KeysWithSuffix(suffix string) (keys []string, err error) {
	var infos []os.FileInfo

	if infos, err = ioutil.ReadDir(fs.Path); err == nil {
		for _, info := range infos {
			if info.IsDir() == false && strings.HasSuffix(info.Name(), suffix) == true {
				keys = append(keys, info.Name())
			}
		}
	}

	return
}

func (fs *fsStore) filePathToFile(file string) string {
	return filepath.Join(fs.Path, sanitizeFilename(file))
}

type storer struct {
	Store
}

func (st *storer) SetString(key string, value string) error {
	return st.Set(key, []byte(value))
}

func (st *storer) GetString(key string) (string, error) {
	b, err := st.Get(key)
	return string(b), err
}

func (st *storer) KeyPair() (KeyPair, error) {
	var kp KeyPair
	b, err := st.Get("keypair")
	if err != nil {
		return kp, err
	}

	err = json.Unmarshal(b, &kp)

	return kp, err
}

func (st *storer) SaveKeyPair(kp KeyPair) error {
	b, err := json.Marshal(&kp)
	if err != nil {
		return err
	}

	return st.Set("keypair", b)
}

func (st *storer) DeleteKeyPair(name string) error {
	return st.Delete("keypair")
}

// Pairing returns the pairing with the given name.
func (st *storer) Pairing(name string) (Pairing, error) {
	return st.pairingForKey(keyForPairingName(name))
}

// SavePairing saves the given pairing.
func (st *storer) SavePairing(pairing Pairing) error {
	b, err := json.Marshal(&pairing)
	if err != nil {
		return err
	}

	return st.Set(keyForPairingName(pairing.Name), b)
}

// DeletePairing deletes the pairing with a given name.
func (st *storer) DeletePairing(name string) error {
	return st.Delete(keyForPairingName(name))
}

// Pairings returns all known pairings.
func (st *storer) Pairings() []Pairing {
	var arr []Pairing
	if ks, err := st.KeysWithSuffix(".pairing"); err == nil {
		for _, k := range ks {
			if p, err := st.pairingForKey(k); err == nil {
				arr = append(arr, p)
			}
		}
	}

	return arr
}

// eneity is used in older versions to store public & private keys
// of the accessory and pairings clients.
// Use Keypair and Pairing instead.
type entity struct {
	Name       string
	PublicKey  []byte
	PrivateKey []byte
}

func (st *storer) pairingForKey(key string) (p Pairing, err error) {
	var b []byte
	if b, err = st.Get(key); err == nil {
		err = json.Unmarshal(b, &p)
	}
	return
}

func (st *storer) entityForKey(key string) (e entity, err error) {
	var b []byte

	if b, err = st.Get(key); err == nil {
		err = json.Unmarshal(b, &e)
	}

	return
}

func keyForName(s string) string {
	return hex.EncodeToString([]byte(s)) + ".entity"
}

func keyForPairingName(s string) string {
	return hex.EncodeToString([]byte(s)) + ".pairing"
}

// sanitizeFilename returns a valid file name by removing invalidcharacters (e.g. colon ":" which is not allowed in file names on Window)
func sanitizeFilename(filename string) string {
	return strings.Replace(filename, ":", "", -1)
}
