package main

import (
	//"encoding/json"
	"database/sql"
	"encoding/json"
	"strconv"

	//"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"

	//"time"

	//	"os/signal"
	"sync/atomic"
	//se usa despues

	//"syscall"
	"github.com/williballenthin/govt"

	//se usa despues
	"github.com/joho/godotenv"

	_ "github.com/lib/pq"
	//"os/exec"
)

//Variables que se van a usar en el server.

type Code_QR struct {
	CODE_QR string `json:"code_qr"`
}

type analizadoresURL struct {
	VIRUSTOTAL           bool
	GOOGLE_SAFE_BROWSING bool
	IP_QUALITY_SCORE     bool
}

type Dispositivo struct {
	AndroidID            string   `json:"android_id"`
	VersionSecurityPatch string   `json:"version_security_patch"`
	VersionSdkInt        int      `json:"version_sdk_int"`
	VersionRelease       string   `json:"version_release"`
	VersionPreviewSdkInt int      `json:"version_preview_sdk_int"`
	VersionIncremental   string   `json:"version_incremental"`
	VersionCodename      string   `json:"version_codename"`
	VersionBaseOS        string   `json:"version_base_os"`
	Board                string   `json:"board"`
	Bootloader           string   `json:"bootloader"`
	Brand                string   `json:"brand"`
	Device               string   `json:"device"`
	Display              string   `json:"display"`
	Fingerprint          string   `json:"fingerprint"`
	Hardware             string   `json:"hardware"`
	Host                 string   `json:"host"`
	ID                   string   `json:"id"`
	Manufacturer         string   `json:"manufacturer"`
	Model                string   `json:"model"`
	Product              string   `json:"product"`
	Supported32BitAbis   []string `json:"supported_32_bit_abis"`
	Supported64BitAbis   []string `json:"supported_64_bit_abis"`
	Type                 string   `json:"type"`
	IsPhysicalDevice     bool     `json:"is_physical_device"`
	SystemFeatures       []string `json:"system_features"`
	SerialNumber         string   `json:"serial_number"`
	IsLowRamDevice       bool     `json:"is_low_ram_device"`
}

type Localizacion struct {
	ID          int     `json:"id"`
	Latitud     float64 `json:"latitud"`
	Longitud    float64 `json:"longitud"`
	Descripcion string  `json:"descripcion"`
}

const (
	imagenBase = "android_base.qcow2"
	prefijo    = "clone_"
	qemuBinary = "qemu-system-x86_64"
)

var bloqueMontaje uint64 = 0
var cloneID uint64
var path string
var db *sql.DB

// Clonado de la bbdd
func getNewIdClone() uint64 {
	return atomic.AddUint64(&cloneID, 1)
}

func newBloqueMontaje() uint64 {
	limBloques, err := strconv.ParseUint(os.Getenv("BLOQUES"), 10, 64)

	fmt.Printf("%s = %d", os.Getenv("BLOQUES"), limBloques)

	if err != nil {
		fmt.Errorf("Error al transformar string a int: %v", err)
	}

	bloqueMontaje = (bloqueMontaje + 1) % limBloques

	return bloqueMontaje
}

func crearClone() (string, error) {
	cloneID = getNewIdClone()
	imagenClone := fmt.Sprintf(prefijo+"%d.qcow2", cloneID)
	cmd := exec.Command("qemu-img", "create", "-f", "qcow2", "-b", imagenBase, "-F", "qcow2", imagenClone)
	cmd.Dir = path

	var e error = nil
	err := cmd.Run()

	if err != nil {
		imagenClone = ""
		e = fmt.Errorf("Error al crear el clone %v", err)
	}

	return imagenClone, e
}

// Start la maquina virtual
func startVM(cloneImage, vncPort string) (*exec.Cmd, error) {

	fmt.Println("Imagen clone: %s,  puerto : %s", cloneImage, vncPort)

	cmd := exec.Command("qemu-system-x86_64",
		"-enable-kvm",
		"-m", "2048",
		"-boot", "c",
		"-vga", "qxl",
		"-hda", cloneImage,
		"-display", "none",
		"-vnc", vncPort,
	)

	cmd.Dir = path

	err := cmd.Start()

	fmt.Println("Emos pasado el cmd.Start")
	rt := cmd
	var e error = nil
	if err != nil {
		rt = nil
		e = fmt.Errorf("Error al iniciar VM: %v", err)
	}

	return rt, e
}

func compartirQr(qr string, cloneImage string) {

	bloque := newBloqueMontaje()
	pathBloqueMontado := "/dev/nbd" + strconv.FormatUint(bloque, 10)
	pathCloneImage := path + "/" + cloneImage

	fmt.Printf("Me lo pides tu qemu? \n")

	fmt.Printf("path bloque mnbd: %s \n path imagen clone: %s \n", pathBloqueMontado, pathCloneImage)

	cmdNbd := exec.Command("sudo",
		"qemu-nbd",
		"-c", pathBloqueMontado, pathCloneImage,
	)

	err := cmdNbd.Run()

	if err != nil {
		fmt.Errorf("Error al intentar conectar la imagen a /dev/nbd%d : %v", bloque, err)
	}

	fmt.Printf("Clone name en compartir = %s \n", cloneImage)

	var newDir = "/mnt/" + strings.TrimSuffix(cloneImage, ".qcow2")

	cmdMkdir := exec.Command("sudo",
		"mkdir", newDir,
	)

	err = cmdMkdir.Run()

	if err != nil {
		fmt.Errorf("Error al crear directorio con sudo mkdir %s : %v", newDir, err)
	}

	fmt.Printf("path montado: %s \n path donde se monta: %s \n", pathBloqueMontado+"p1", newDir)

	cmdMount := exec.Command("sudo",
		"mount", pathBloqueMontado+"p1", newDir,
	)

	err = cmdMount.Run()

	if err != nil {
		fmt.Errorf("Error al montar el disco virtual: %v", err)
	}

	fmt.Printf("%s \n", newDir)
	fmt.Printf("echo '%s' > %s/android-9.0-r2/data/qr_code.txt\n", qr, newDir)

	qrPath := fmt.Sprintf("%s/android-9.0-r2/data/qr_code.txt", newDir)
	err = os.WriteFile(qrPath, []byte(qr), 0644)

	/*cmdCompartir := exec.Command(
		fmt.Sprintf("echo '%s' > %s/android-9.0-r2/data/qr_code.txt", qr, newDir),
	)

	err = cmdCompartir.Run()*/
	if err != nil {
		fmt.Errorf("Error al compartir el qr en %s: %v", qrPath, err)
	}

	//mnt/android_disk/android-9.0-r2/data/

	fmt.Printf("umount %s \n", newDir)

	cmdDesmontar := exec.Command("sudo",
		"umount", newDir,
	)

	err = cmdDesmontar.Run()

	if err != nil {
		fmt.Errorf("Error al compartir el qr en %s/dev/qr_code.txt: %v", newDir, err)
	}

	fmt.Printf("qemu-nbd %s \n", pathBloqueMontado)

	cmdDesconectar := exec.Command("sudo",
		"qemu-nbd",
		"-d", pathBloqueMontado,
	)

	err = cmdDesconectar.Run()

	if err != nil {
		fmt.Errorf("Error al desconectar el bloque %s: %v", pathBloqueMontado, err)
	}

}

// Elimina la maquina clone
func deleteClone(cloneImage string) error {
	clonepath := path + "/" + cloneImage
	return os.Remove(clonepath)
}

// consulta segura. Proceso de creación y puesta a punto de la maquina virtual.
func consultaSegura(w http.ResponseWriter, r *http.Request) {

	fmt.Println("Putamadre llego PREPOST")

	if r.Method != "POST" {
		http.Error(w, "Metodo http no valido", http.StatusMethodNotAllowed)
	}

	fmt.Println("Putamadre llego postPost")

	fmt.Fprintln(w, "Putamadre llego")

	var qrcode Code_QR
	decode := json.NewDecoder(r.Body)
	err := decode.Decode(&qrcode)

	if err != nil {
		http.Error(w, "Error al procesar el JSON", http.StatusBadRequest)
	}

	fmt.Println(qrcode.CODE_QR)

	path = os.Getenv("CLONE_PATH")

	cloneName, err := crearClone()
	fmt.Printf("Nombre de clone puta madre: %s \n", cloneName)

	if err != nil {
		fmt.Errorf("Error en clonado main: %v", err)
	}

	// antes de empezar a mostrar tengo que hacer "sudo modprobe nbd" para habilitar los bloques.
	compartirQr(qrcode.CODE_QR, cloneName)

	/*vmCmd*/
	_, err = startVM(cloneName, ":1")

	if err != nil {
		fmt.Errorf("Error al iniciar VM main: %v", err)
	}

	/*fmt.Println("VM running. Press ENTER to stop...")
	fmt.Scanln()

	if err := vmCmd.Process.Kill(); err != nil {
		fmt.Println("Error stopping VM:", err)
	}

	time.Sleep(2 * time.Second)
	if err := deleteClone(cloneName); err != nil {
		fmt.Println("Error deleting clone:", err)
	} else {
		fmt.Println("Clone deleted.")
	}*/

}

func finConsultaSegura(w http.ResponseWriter, r *http.Request) {
	/*
		decode := json.NewDecoder(r.Body)
		err := decode.Decode(&qrcode)

		if err != nil {
			http.Error(w, "Error al procesar el JSON", http.StatusBadRequest)
		}*/
}

// Apartado de analisis de QR por via externa.
func analisisURL(url string) map[string]bool {

	//Diccionario de salidas de los analizadores.

	dicAnalsizadores := make(map[string]bool)

	dicAnalsizadores["virustotal"] = analisisVirusTotal(url)

	return dicAnalsizadores
}

func analisisVirusTotal(url string) bool {
	vt_api_key := os.Getenv("VIRUSTOTAL_KEY")
	apiURL := "https://www.virustotal.com/vtapi/v2/"

	c, err := govt.New(govt.SetApikey(vt_api_key), govt.SetUrl(apiURL))

	if err != nil {
		log.Println("Error al hacer el govt VT: ", err)
	}

	r, err := c.GetUrlReport(url)

	if err != nil {
		log.Println("Error al hacer el report VT: ", err)
	}

	//j, err := json.MarshalIndent(r, "", "    ")

	if err != nil {
		log.Println("Error al hacer el json? VT: ", err)
	}

	fmt.Println("UslREport: ", r.Positives)

	return r.Positives > 0
}

// Parte de consulta a base de datos propia.
func crearBBDD() {

	query := `CREATE TABLE IF NOT EXISTS qrs (
		id SERIAL PRIMARY KEY,
		contenido TEXT NOT NULL UNIQUE
	)`

	_, err := db.Exec(query)

	if err != nil {
		log.Fatal("Error al crear la tabla qrs:", err)
	}

	query = `CREATE TABLE IF NOT EXISTS dispositivo (
		android_id TEXT PRIMARY KEY,
		version_security_patch TEXT,
		version_sdk_int INTEGER,
		version_release TEXT,
		version_preview_sdk_int INTEGER,
		version_incremental TEXT,
		version_codename TEXT,
		version_base_os TEXT,
		board TEXT,
		bootloader TEXT,
		brand TEXT,
		device TEXT,
		display TEXT,
		fingerprint TEXT,
		hardware TEXT,
		host TEXT,
		id TEXT,
		manufacturer TEXT,
		model TEXT,
		product TEXT,
		supported_32_bit_abis TEXT[],
		supported_64_bit_abis TEXT[],
		type TEXT,
		is_physical_device BOOLEAN,
		system_features TEXT[],
		serial_number TEXT,
		is_low_ram_device BOOLEAN
		);`

	_, err = db.Exec(query)

	if err != nil {
		log.Fatal("Error al crear la tabla dispositivo:", err)
	}

	query = `CREATE TABLE IF NOT EXISTS localizacion (
		id SERIAL PRIMARY KEY,
		latitud DOUBLE PRECISION NOT NULL,
		longitud DOUBLE PRECISION NOT NULL,
		descripcion TEXT,
		UNIQUE(latitud, longitud)
		);`

	_, err = db.Exec(query)

	if err != nil {
		log.Fatal("Error al crear la tabla localizacion:", err)
	}

	query = `CREATE TABLE IF NOT EXISTS dispositivo_qr (
		qr_id INTEGER REFERENCES qrs(id) ON DELETE CASCADE,
		android_id TEXT REFERENCES dispositivo(android_id) ON DELETE CASCADE,
		PRIMARY KEY (qr_id, android_id)
	);`

	_, err = db.Exec(query)

	if err != nil {
		log.Fatal("Error al crear la tabla dispositivo_qr:", err)
	}

	query = `CREATE TABLE IF NOT EXISTS qr_localizacion (
		qr_id INTEGER REFERENCES qrs(id) ON DELETE CASCADE,
		localizacion_id INTEGER REFERENCES localizacion(id) ON DELETE CASCADE,
		PRIMARY KEY (qr_id, localizacion_id)
	);`

	_, err = db.Exec(query)

	if err != nil {
		log.Fatal("Error al crear la tabla qr_localizacion:", err)
	}

	/*query = `CREATE TABLE IF NOT EXISTS ip_disp (
		id
	);`

	_, err = db.Exec(query)

	if err != nil {
		log.Fatal("Error al crear la tabla:", err)
	}*/

	fmt.Println("Tabla creada exitosamente")
}

func consultaBBDD(qr Code_QR) bool {

	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")

	db, err := sql.Open("postgres",
		fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname))

	if err != nil {
		log.Fatal(err)
		defer db.Close()
	}

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	//sql.Open("postgres",
	// "postgres://admin_tfg:admin_tfg@localhost:5432/db_tfg?sslmode=disable")
	fmt.Println("Successfully connected to PostgreSQL!")

	rows, err := db.Query("Select * from qrs where contenido = $1", qr.CODE_QR)

	if err != nil {
		log.Print("Error en la query: %s", err)
	}
	rows.Close()

	var rt bool = false

	if rows.Next() {
		rt = true
	}

	//esto va despues del analizador.
	/*if (rt) {
		_ , err := db.Exec("INSERT INTO qrs (contenido) VALUES (%s)", qr.CODE_QR)
		if err != nil {
			log.Print("Error al insertar el valor: %s", err)
		}
	}*/

	return rt
}

func guardar_disp(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Metodo http no valido", http.StatusMethodNotAllowed)
	} else {

		var datos Dispositivo
		decode := json.NewDecoder(r.Body)
		err := decode.Decode(&datos)

		//Empezamos transaccion

		if err != nil {
			http.Error(w, fmt.Sprintf("Error al procesar el JSON: %s", err), http.StatusBadRequest)
			log.Fatal(err)

		} else {
			//fmt.Println(datos)

			if err != nil {
				http.Error(w, "Error al procesar la información", http.StatusInternalServerError)
				log.Fatal(err)
			}

			query := `
			INSERT INTO dispositivo (
			android_id,
			version_security_patch,
			version_sdk_int,
			version_release,
			version_preview_sdk_int,
			version_incremental,
			version_codename,
			version_base_os,
			board,
			bootloader,
			brand,
			device,
			display,
			fingerprint,
			hardware,
			host,
			id,
			manufacturer,
			model,
			product,
			type,
			is_physical_device,
			serial_number,
			is_low_ram_device
			) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
			$21, $22, $23, $24
			);
			`

			// Asegúrate de que tienes la estructura "Dispositivo" y los datos decodificados en la variable "datos"
			_, err := db.Exec(
				query,
				datos.AndroidID,
				datos.VersionSecurityPatch,
				datos.VersionSdkInt,
				datos.VersionRelease,
				datos.VersionPreviewSdkInt,
				datos.VersionIncremental,
				datos.VersionCodename,
				datos.VersionBaseOS,
				datos.Board,
				datos.Bootloader,
				datos.Brand,
				datos.Device,
				datos.Display,
				datos.Fingerprint,
				datos.Hardware,
				datos.Host,
				datos.ID,
				datos.Manufacturer,
				datos.Model,
				datos.Product,
				datos.Type,
				datos.IsPhysicalDevice,
				datos.SerialNumber,
				datos.IsLowRamDevice,
			)

			if err != nil {
				fmt.Errorf("Error al insertar datos del dispositivo: %v", err)
				// Aquí puedes hacer rollback si estás en una transacción
				return
			}

			if err != nil {
				fmt.Errorf("error al insertar datos: %v", err)
			}

			rows, err := db.Query("SELECT android_id FROM dispositivo")

			if err != nil {
				log.Fatalf("Error al hacer el SELECT: %v", err)
			}
			defer rows.Close()

			for rows.Next() {
				var androidID string

				// IMPORTANTE: El orden debe coincidir con las columnas en la tabla (sin los TEXT[])
				err := rows.Scan(
					&androidID,
				)

				if err != nil {
					log.Printf("Error al escanear fila: %v", err)
					continue
				}

				fmt.Printf("Dispositivo: %s \n", androidID)
			}

		}
	}
}

func guardar_qr(qr Code_QR, android_id string, localizacion Localizacion) {

	tx, err := db.Begin()

	if err != nil {
		fmt.Errorf("Error al insertar el QR: %v", err)
		log.Fatal(err)
	}

	insert_qr := `INSERT INTO qrs (contenido) VALUES ($1) RETURNING id;`

	var qrID int
	err = tx.QueryRow(insert_qr, qr.CODE_QR).Scan(&qrID)
	if err != nil {
		fmt.Errorf("Error al insertar el QR: %v", err)
		tx.Rollback()
	}

	relacion_qr_disp := `INSERT INTO dispositivo_qr (qr_id, android_id) VALUES ($1, $2);`
	_, err = tx.Exec(relacion_qr_disp, qrID, android_id)

	if err != nil {
		fmt.Errorf("Error al crear la relación entre el QR y el dispositivo: %v", err)
		tx.Rollback()
	}

	err = tx.Commit()
	if err != nil {
		fmt.Errorf("Error al hacer commit de la transacción: %v", err)
		tx.Rollback()
	}

	fmt.Println("QR insertado y relacionado con el dispositivo correctamente.")
}

//Analisis del QR.
//Engloba todo el proceso.

func analisisQR(w http.ResponseWriter, r *http.Request) {

	fmt.Println("Entro en la api")

	if r.Method != "POST" {
		http.Error(w, "Metodo http no valido", http.StatusMethodNotAllowed)
	} else {
		var datos map[string]string

		decode := json.NewDecoder(r.Body)
		err := decode.Decode(&datos)

		var cd_qr Code_QR
		cd_qr.CODE_QR = datos["code_qr"]

		androidID := datos["android_id"]

		fmt.Println(androidID)
		fmt.Println(cd_qr.CODE_QR)

		if err != nil {
			http.Error(w, "Error al procesar el JSON", http.StatusBadRequest)
		} else {
			fmt.Println(cd_qr.CODE_QR)

			consultaBBDD(cd_qr)

			//Salida con los datos del analisis externo.
			dicAnalisis := analisisURL(cd_qr.CODE_QR)

			dicFinal := []map[string]string{}

			for analizador, resultado := range dicAnalisis {

				var resultStrign string = "no se sabe"

				if resultado {
					resultStrign = "Peligroso"
					guardar_qr(cd_qr,androidID,Localizacion{})
				} else {
					resultStrign = "No peligroso"
				}

				nuevo := map[string]string{
					"analizador": analizador,
					"resultado":  resultStrign,
				}

				dicFinal = append(dicFinal, nuevo)
			}

			w.WriteHeader(http.StatusCreated)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(dicFinal)

		}
	}
}

//Main del server

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error cargando .env")
	}

	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")

	db, err = sql.Open("postgres",
		fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname))

	if err != nil {
		log.Fatal(err)
		defer db.Close()
	}

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	crearBBDD()

	http.HandleFunc("/analisis_qr", analisisQR)
	http.HandleFunc("/consulta_segura", consultaSegura)
	http.HandleFunc("/cerrar_sesion_consulta_segura", finConsultaSegura)
	http.HandleFunc("/guardar_disp", guardar_disp)

	fmt.Println("Servidor escuchando en http://0.0.0.0:80")
	if err := http.ListenAndServe(":80", nil); err != nil {
		log.Fatal(err)
	}

}

/*
func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error cargando .env")
	}

	path = os.Getenv("CLONE_PATH")

	cloneImage, err := crearClone()
	if err != nil {
		fmt.Println("Error creating clone:", err)
		return
	}

	fmt.Println("Clone created:", cloneImage)
	vmCmd, err := startVM(cloneImage, ":1")
	if err != nil {
		fmt.Println("Error starting VM:", err)
		return
	}

	fmt.Println("VM running. Press ENTER to stop...")
	fmt.Scanln()

	if err := vmCmd.Process.Kill(); err != nil {
		fmt.Println("Error stopping VM:", err)
	}

	time.Sleep(2 * time.Second)
	if err := deleteClone(cloneImage); err != nil {
		fmt.Println("Error deleting clone:", err)
	} else {
		fmt.Println("Clone deleted.")
	}
}
*/
