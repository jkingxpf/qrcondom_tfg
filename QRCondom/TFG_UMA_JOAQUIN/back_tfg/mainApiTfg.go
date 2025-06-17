package main
 
import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/williballenthin/govt"
)

//Variables que se van a usar en el server.

type Code_QR struct {
	CODE_QR string `json:"code_qr"`
}

type Dispositivo struct {
	AndroidID            string   `json:"androidId"`
	VersionSecurityPatch string   `json:"version.securityPatch"`
	VersionSdkInt        int      `json:"version.sdkInt"`
	VersionRelease       string   `json:"version.release"`
	VersionPreviewSdkInt int      `json:"version.previewSdkInt"`
	VersionIncremental   string   `json:"version.incremental"`
	VersionCodename      string   `json:"version.codename"`
	VersionBaseOS        string   `json:"version.baseOS"`
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
	Supported32BitAbis   []string `json:"supported32BitAbis"`
	Supported64BitAbis   []string `json:"supported64BitAbis"`
	Type                 string   `json:"type"`
	IsPhysicalDevice     bool     `json:"isPhysicalDevice"`
	SystemFeatures       []string `json:"systemFeatures"`
	SerialNumber         string   `json:"serialNumber"`
	IsLowRamDevice       bool     `json:"isLowRamDevice"`
}

type Localizacion struct {
	Latitud  float64 `json:"latitude"`
	Longitud float64 `json:"longitude"`
}

type DatosAsociadosAndroid struct {
	cloneID uint64
	puerto  uint32
	vmCmd   *exec.Cmd
}

const (
	imagenBase = "android_base.qcow2"
	prefijo    = "clone_"
	qemuBinary = "qemu-system-x86_64"
)

var path string
var db *sql.DB
var androidDicc map[string]DatosAsociadosAndroid = make(map[string]DatosAsociadosAndroid)
var bloqueMontaje uint64 = 0
var cloneID uint64 = 0
var puerto uint32 = 0

func getNewIdClone() uint64 {
	return atomic.AddUint64(&cloneID, 1)
}

func getNewPuerto() uint32 {
	return atomic.AddUint32(&puerto, 1)
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

func crearClone(android_id string) (string, error) {
	cloneID = androidDicc[android_id].cloneID
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

func startVM(cloneImage, vncPort string) (*exec.Cmd, error) {

	fmt.Printf("Imagen clone: %s,  puerto : %s \n", cloneImage, vncPort)

	cmd := exec.Command("qemu-system-x86_64",
		"-enable-kvm",
		"-m", "4096",
		"-boot", "c",
		"-vga", "qxl",
		"-hda", cloneImage,
		"-display", "none",
		"-vnc", vncPort,
	)

	cmd.Dir = path

	err := cmd.Start()

	rt := cmd
	var e error = nil
	if err != nil {
		rt = nil
		e = fmt.Errorf("Error al iniciar VM: %v", err)
	}

	return rt, e
}

func compartirQr(qr string, cloneImage string) bool {

	var fallo bool = false

	bloque := newBloqueMontaje()
	pathBloqueMontado := "/dev/nbd" + strconv.FormatUint(bloque, 10)
	pathCloneImage := path + "/" + cloneImage

	fmt.Printf("path bloque mnbd: %s \n path imagen clone: %s \n", pathBloqueMontado, pathCloneImage)

	cmdNbd := exec.Command("sudo",
		"qemu-nbd",
		"-c", pathBloqueMontado, pathCloneImage,
	)

	err := cmdNbd.Run()

	if err != nil {
		fmt.Errorf("Error al intentar conectar la imagen a /dev/nbd%d : %v", bloque, err)
		fallo = true
	}

	fmt.Printf("Clone name en compartir = %s \n", cloneImage)

	var newDir = "/mnt/" + strings.TrimSuffix(cloneImage, ".qcow2")

	cmdMkdir := exec.Command("sudo",
		"mkdir", newDir,
	)

	err = cmdMkdir.Run()

	if err != nil {
		fmt.Printf("Error al crear directorio con sudo mkdir %s : %v", newDir, err)
		fallo = true
	}

	fmt.Printf("path montado: %s \n path donde se monta: %s \n", pathBloqueMontado+"p1", newDir)

	cmdMount := exec.Command("sudo",
		"mount", pathBloqueMontado+"p1", newDir,
	)

	err = cmdMount.Run()

	if err != nil {
		fmt.Printf("Error al montar el disco virtual: %v", err)
		fallo = true
	}

	qrPath := fmt.Sprintf("%s/android-9.0-r2/data/qr_code.txt", newDir)
	err = os.WriteFile(qrPath, []byte(qr), 0644)

	if err != nil {
		fmt.Printf("Error al compartir el qr en %s: %v", qrPath, err)
		fallo = true
	}

	cmdDesmontar := exec.Command("sudo",
		"umount", newDir,
	)

	err = cmdDesmontar.Run()

	if err != nil {
		fmt.Printf("Error al compartir el qr en %s/dev/qr_code.txt: %v", newDir, err)
		fallo = true
	}

	fmt.Printf("qemu-nbd %s \n", pathBloqueMontado)

	cmdDesconectar := exec.Command("sudo",
		"qemu-nbd",
		"-d", pathBloqueMontado,
	)

	err = cmdDesconectar.Run()

	if err != nil {
		fmt.Printf("Error al desconectar el bloque %s: %v", pathBloqueMontado, err)
		fallo = true
	}

	cmdrm := exec.Command("sudo",
		"rm", "-r", newDir,
	)

	err = cmdrm.Run()

	if err != nil {
		fmt.Printf("Error al eliminar con rm %s : %v", newDir, err)
		fallo = true
	}

	return fallo

}

func deleteClone(cloneImage string) error {
	clonepath := path + "/" + cloneImage
	return os.Remove(clonepath)
}

func consultaSegura(w http.ResponseWriter, r *http.Request) {

	if r.Method != "POST" {
		http.Error(w, "Metodo http no valido", http.StatusMethodNotAllowed)
	}

	var qrcode Code_QR

	var dic map[string]string
	decode := json.NewDecoder(r.Body)
	err := decode.Decode(&dic)

	qrcode.CODE_QR = dic["qr_code"]
	var datosAsos DatosAsociadosAndroid

	datosAsos.cloneID = getNewIdClone()
	datosAsos.puerto = getNewPuerto()

	androidDicc[dic["android_id"]] = datosAsos

	if err != nil {
		http.Error(w, "Error al procesar el JSON", http.StatusBadRequest)
	}

	fmt.Println(qrcode.CODE_QR)

	path = os.Getenv("CLONE_PATH")

	cloneName, err := crearClone(dic["android_id"])

	if err != nil {
		http.Error(w, "Error al clonar", http.StatusBadRequest)
	}

	if compartirQr(qrcode.CODE_QR, cloneName) {
		http.Error(w, "Error al compartir", http.StatusBadRequest)
	}

	port := fmt.Sprintf(":%d",
		androidDicc[dic["android_id"]].puerto)

	datosAsos.vmCmd, err = startVM(cloneName, port)

	androidDicc[dic["android_id"]] = datosAsos

	if err != nil {
		fmt.Println(err)
		http.Error(w, "Error al iniciar la VM", http.StatusBadRequest)
	}

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(androidDicc[dic["android_id"]].puerto)
}

func finConsultaSegura(w http.ResponseWriter, r *http.Request) {

	var respuesta map[string]string
	decode := json.NewDecoder(r.Body)
	err := decode.Decode(&respuesta)

	if err != nil {
		http.Error(w, "Error al procesar el JSON", http.StatusBadRequest)
	}

	infoAsoc := androidDicc[respuesta["android_id"]]

	if infoAsoc.vmCmd == nil {
		http.Error(w, "Error ¿poque null?", http.StatusBadRequest)
	}

	err = infoAsoc.vmCmd.Process.Kill()

	if err != nil {
		http.Error(w, "Error parando VM:", http.StatusBadRequest)
	}

	imagenClone := fmt.Sprintf(prefijo+"%d.qcow2", infoAsoc.cloneID)

	err = deleteClone(imagenClone)
	if err != nil {
		http.Error(w, "Error deleting clone:", http.StatusBadRequest)
		fmt.Println("Error deleting clone:", err)
	} else {
		fmt.Println("Clone deleted.")
	}

	delete(androidDicc, respuesta["android_id"])

	fmt.Println("Eliminacion completada X_X")

}

func analisisURL(urlObjetivo string) map[string]bool {

	dicAnalsizadores := make(map[string]bool)

	dicAnalsizadores["virustotal"] = analisisVirusTotal(urlObjetivo)
	dicAnalsizadores["urlhaus"] = analisisURLhaus(urlObjetivo)
	dicAnalsizadores["PhishTank"] = analisisPhishTank(urlObjetivo)

	return dicAnalsizadores
}

func analisisPhishTank(urlObjetivo string) bool {

	form := url.Values{}
	fmt.Printf("URL obejtivo = %s \n", urlObjetivo)
	form.Add("url", urlObjetivo)
	form.Add("format", "json")

	req, _ := http.NewRequest("POST", "https://checkurl.phishtank.com/checkurl/", bytes.NewBufferString(form.Encode()))

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "phishtank/username")

	cliente := &http.Client{}
	resp, _ := cliente.Do(req)

	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error al leer el cuerpo:", err)
		return false
	}

	var bodyResp map[string]json.RawMessage
	if err := json.Unmarshal(bodyBytes, &bodyResp); err != nil {
		fmt.Println("Error al decodificar JSON:", err)
		return false
	}

	var respuesta map[string]bool
	json.Unmarshal(bodyResp["results"], &respuesta)
	fmt.Printf("%+v\n", respuesta)

	return respuesta["in_database"]
}

func analisisURLhaus(urlObjetivo string) bool {

	urlhaus_key := os.Getenv("URLHAUS_KEY")

	form := url.Values{}
	form.Add("url", urlObjetivo)

	req, _ := http.NewRequest("POST", "https://urlhaus-api.abuse.ch/v1/url/", bytes.NewBufferString(form.Encode()))

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Auth-key", urlhaus_key)

	cliente := &http.Client{}
	resp, _ := cliente.Do(req)

	defer resp.Body.Close()

	var bodyResp map[string]string
	json.NewDecoder(resp.Body).Decode(&bodyResp)

	return bodyResp["query_status"] == "ok"
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

	if err != nil {
		log.Println("Error al hacer el json? VT: ", err)
	}

	return r.Positives > 0
}

func conexionBBDD() {

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

}

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
		qr_id INTEGER REFERENCES qrs(id) ,
		android_id TEXT REFERENCES dispositivo(android_id) ,
		PRIMARY KEY (qr_id, android_id)
	);`

	_, err = db.Exec(query)

	if err != nil {
		log.Fatal("Error al crear la tabla dispositivo_qr:", err)
	}

	query = `CREATE TABLE IF NOT EXISTS qr_localizacion (
		qr_id INTEGER REFERENCES qrs(id) ,
		localizacion_id INTEGER REFERENCES localizacion(id) ,
		PRIMARY KEY (qr_id, localizacion_id)
	);`

	_, err = db.Exec(query)

	if err != nil {
		log.Fatal("Error al crear la tabla qr_localizacion:", err)
	}

	fmt.Println("Tabla creada exitosamente")
}

func consultaBBDD(qr Code_QR) bool {

	rows, err := db.Query("Select * from qrs where contenido = $1", qr.CODE_QR)

	if err != nil {
		log.Print("Error en la query: %s", err)
	}
	rt := rows.Next()
	rows.Close()

	return rt
}

func guardar_disp(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Metodo http no valido", http.StatusMethodNotAllowed)
	} else {

		var datosJson map[string]json.RawMessage
		decode := json.NewDecoder(r.Body)
		err := decode.Decode(&datosJson)

		var datos Dispositivo
		json.Unmarshal(datosJson["Dispositivo"], &datos)

		if err != nil {
			http.Error(w, fmt.Sprintf("Error al procesar el JSON: %s", err), http.StatusBadRequest)

		} else {

			rows, err := db.Query("SELECT android_id FROM dispositivo")
			if err != nil {
				http.Error(w, fmt.Sprintf("Error al hacer el SELECT: %s", err), http.StatusBadRequest)
			}
			defer rows.Close()

			if !rows.Next() {
				fmt.Println("Entro en insertar")
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

				_, err = db.Exec(
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
					http.Error(w, fmt.Sprintf("Error al hacer el INSERT del dispositivo: %s", err), http.StatusBadRequest)
				}
			}
		}
	}
}

func guardar_qr(qr Code_QR, android_id string, localizacion Localizacion) {

	tx, err := db.Begin()

	if err != nil {
		fmt.Errorf("Error al insertar el QR: %v", err)
	}

	insert_qr := `
		INSERT INTO qrs (contenido)
		VALUES ($1)
		ON CONFLICT (contenido) DO NOTHING
		RETURNING id;
		`

	var qrID int
	err = tx.QueryRow(insert_qr, qr.CODE_QR).Scan(&qrID)
	if err == sql.ErrNoRows {
		err = tx.QueryRow("SELECT id FROM qrs WHERE contenido = $1", qr.CODE_QR).Scan(&qrID)
	}

	if err != nil {
		fmt.Errorf("Error guardar o seleccionar el QR: %v", err)
		tx.Rollback()
	}

	relacion_qr_disp := `INSERT INTO dispositivo_qr (qr_id, android_id) VALUES ($1, $2);`
	_, err = tx.Exec(relacion_qr_disp, qrID, android_id)

	if err != nil {
		fmt.Errorf("Error al crear la relación entre el QR y el dispositivo: %v", err)
		tx.Rollback()
	}

	var localizacionID int
	insert_localizacion := `
	INSERT INTO localizacion (latitud, longitud, descripcion)
	VALUES ($1, $2, $3)
	ON CONFLICT (latitud, longitud) DO UPDATE SET descripcion = EXCLUDED.descripcion
	RETURNING id;
`

	err = tx.QueryRow(insert_localizacion, localizacion.Latitud, localizacion.Longitud, nil).Scan(&localizacionID)
	if err == sql.ErrNoRows {
		err = tx.QueryRow(
			"SELECT id FROM localizacion WHERE latitud = $1 AND longitud = $2",
			localizacion.Latitud, localizacion.Longitud,
		).Scan(&localizacionID)
	}

	if err != nil {
		log.Println("Error al insertar o recuperar localización:", err)
		tx.Rollback()
	}

	relacion_qr_localizacion := `
		INSERT INTO qr_localizacion (qr_id, localizacion_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING;
	`
	_, err = tx.Exec(relacion_qr_localizacion, qrID, localizacionID)

	if err != nil {
		log.Println("Error al insertar en qr_localizacion:", err)
		tx.Rollback()
	}

	err = tx.Commit()
	if err != nil {
		fmt.Errorf("Error al hacer commit de la transacción: %v", err)
		tx.Rollback()
	}
}

func analisisQR(w http.ResponseWriter, r *http.Request) {

	if r.Method != "POST" {
		http.Error(w, "Metodo http no valido", http.StatusMethodNotAllowed)
	} else {
		var datos map[string]json.RawMessage

		decode := json.NewDecoder(r.Body)
		err := decode.Decode(&datos)

		if err != nil {
			http.Error(w, "Error al procesar el JSON de datos", http.StatusBadRequest)
		}

		var cd_qr Code_QR
		err = json.Unmarshal(datos["code_qr"], &cd_qr.CODE_QR)

		if err != nil {
			http.Error(w, "Error al procesar el JSON de QR", http.StatusBadRequest)
		}

		var androidID string

		err = json.Unmarshal(datos["androidId"], &androidID)

		fmt.Println(androidID)

		if err != nil {
			http.Error(w, "Error al procesar el JSON de AndroidId", http.StatusBadRequest)
		}

		var posicion Localizacion
		json.Unmarshal(datos["Localizacion"], &posicion)

		if err != nil {
			http.Error(w, "Error al procesar el JSON Localizacion", http.StatusBadRequest)
		} else {
			dicFinal := []map[string]string{}

			if !consultaBBDD(cd_qr) {
				dicAnalisis := analisisURL(cd_qr.CODE_QR)

				for analizador, resultado := range dicAnalisis {

					var resultStrign string = "no se sabe"

					if resultado {
						resultStrign = "Peligroso"
						guardar_qr(cd_qr, androidID, posicion)
					} else {
						resultStrign = "No peligroso"
					}

					nuevo := map[string]string{
						"analizador": analizador,
						"resultado":  resultStrign,
					}

					dicFinal = append(dicFinal, nuevo)
				}
			} else {
				var enBBDD string = "Peligroso"

				analisisPropio := map[string]string{
					"analizador": "QRCondom",
					"resultado":  enBBDD,
				}
				dicFinal = append(dicFinal, analisisPropio)

			}

			w.WriteHeader(http.StatusCreated)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(dicFinal)
		}
	}
}

func main() {

	conexionBBDD()
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
