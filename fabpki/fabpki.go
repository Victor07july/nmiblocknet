/////////////////////////////////////////////
//    THE BLOCKCHAIN PKI EXPERIMENT     ////
///////////////////////////////////////////
/*
	This is the fabpki, a chaincode that implements a Public Key Infrastructure (PKI)
	for measuring instruments. It runs in Hyperledger Fabric 1.4.
	He was created as part of the PKI Experiment. You can invoke its methods
	to store measuring instruments public keys in the ledger, and also to verify
	digital signatures that are supposed to come from these instruments.

	@author: Wilson S. Melo Jr.
	@date: Oct/2019
*/
package main

import (
	//the majority of the imports are trivial...
	"bytes"
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"strconv"
	"time"

	//these imports are for Hyperledger Fabric interface
	//"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	//sc "github.com/hyperledger/fabric/protos/peer"
	sc "github.com/hyperledger/fabric-protos-go/peer"
)

/* All the following functions are used to implement fabpki chaincode. This chaincode
basically works with 2 main features:
	1) A Register Authority RA (e.g., Inmetro) verifies a new measuring instrument (MI) and attests
	the correspondence between the MI's private key and public key. After doing this, the RA
	inserts the public key into the ledger, associating it with the respective instrument ID.

	2) Any client can ask for a digital signature ckeck. The client informs the MI ID, an
	information piece (usually a legally relevant register) and its supposed digital signature.
	The chaincode retrieves the MI public key and validates de digital signature.
*/

// SmartContract defines the chaincode base structure. All the methods are implemented to
// return a SmartContrac type.
type SmartContract struct {
}

// ECDSASignature represents the two mathematical components of an ECDSA signature once
// decomposed.
type ECDSASignature struct {
	R, S *big.Int
}

// Meter constitutes our key|value struct (digital asset) and implements a single
// record to manage the
// meter public key and measures. All blockchain transactions operates with this type.
// IMPORTANT: all the field names must start with upper case
type Meter struct {
	//PubKey ecdsa.PublicKey `json:"pubkey"`
	PubKey string `json:"pubkey"`
	MyDate string `json:"mydate"`
}

type Trajeto struct {
	//estrutura de dados do trajeto
	Distancia   string `json:"distancia"`
	Combustivel string `json:"combustivel"`
}

type numero struct {
	DIstancia   string `json:"distancia"`
	Combustivel string `json:"combustivel"`
}

type Station struct {
	Timestamp   string `json:"timestamp"`
	Temperature string `json:"temperature"`
	WindSpeed   string `json:"windspeed"`
	Insolation  string `json:"insolation"`
}

type WeatherAPI struct {
	CityName    string `json:"cityname"`
	Situation   string `json:"situation"`
	Temperature string `json:"temperature"`
	// Timestamp   string `json:"timestamp"`
	Date string `json:"date"`
	Hour string `json:"hour"`
}

// PublicKeyDecodePEM method decodes a PEM format public key. So the smart contract can lead
// with it, store in the blockchain, or even verify a signature.
// - pemEncodedPub - A PEM-format public key
func PublicKeyDecodePEM(pemEncodedPub string) ecdsa.PublicKey {
	blockPub, _ := pem.Decode([]byte(pemEncodedPub))
	x509EncodedPub := blockPub.Bytes
	genericPublicKey, _ := x509.ParsePKIXPublicKey(x509EncodedPub)
	publicKey := genericPublicKey.(*ecdsa.PublicKey)

	return *publicKey
}

// Init method is called when the fabpki is instantiated.
// Best practice is to have any Ledger initialization in separate function.
// Note that chaincode upgrade also calls this function to reset
// or to migrate data, so be careful to avoid a scenario where you
// inadvertently clobber your ledger's data!
func (s *SmartContract) Init(stub shim.ChaincodeStubInterface) sc.Response {
	return shim.Success(nil)
}

// Invoke function is called on each transaction invoking the chaincode. It
// follows a structure of switching calls, so each valid feature need to
// have a proper entry-point.
func (s *SmartContract) Invoke(stub shim.ChaincodeStubInterface) sc.Response {
	// extract the function name and args from the transaction proposal
	fn, args := stub.GetFunctionAndParameters()

	//implements a switch for each acceptable function
	if fn == "registerMeter" {
		//registers a new meter into the ledger
		return s.registerMeter(stub, args)

	} else if fn == "checkSignature" {
		//inserts a measurement which increases the meter consumption counter. The measurement
		return s.checkSignature(stub, args)

	} else if fn == "sleepTest" {
		//retrieves the accumulated consumption
		return s.sleepTest(stub, args)

	} else if fn == "countHistory" {
		//look for a specific fill up record and brings its changing history
		return s.countHistory(stub, args)

	} else if fn == "countLedger" {
		//look for a specific fill up record and brings its changing history
		return s.countLedger(stub)

	} else if fn == "queryLedger" {
		//execute a CouchDB query, args must include query expression
		return s.queryLedger(stub, args)

	} else if fn == "EnumerateHistory" {
		//execute a CouchDB query, args must include query expression
		return s.EnumerateHistory(stub, args)
	} else if fn == "Mynum" {
		// função da Stephanie
		return s.Mynum(stub, args)
	} else if fn == "registerWeatherFromWeb" {
		// register api weather info
		return s.registerWeatherFromWeb(stub, args)
	} else if fn == "getWeatherFromWeb" {
		// gets api weather info
		return s.getWeatherFromWeb(stub, args)
	} else if fn == "queryWebWeatherHistory" {
		// gets weather from past
		return s.queryWebWeatherHistory(stub, args)
	}

	//function fn not implemented, notify error
	return shim.Error("Chaincode does not support this function.")
}

/*
SmartContract::registerMeter(...)
Does the register of a new meter into the ledger.
The meter is the base of the key|value structure.
The key constitutes the meter ID.
- args[0] - meter ID
- args[1] - the public key associated with the meter
*/
func (s *SmartContract) registerMeter(stub shim.ChaincodeStubInterface, args []string) sc.Response {

	//validate args vector lenght
	if !(len(args) == 2 || len(args) == 3) {
		return shim.Error("It was expected the parameters: <meter id> <public key> [encrypted inital consumption]")
	}

	//gets the parameters associated with the meter ID and the public key (in PEM format)
	meterid := args[0]
	strpubkey := args[1]

	//creates the meter record with the respective public key
	var meter = Meter{PubKey: strpubkey}

	//encapsulates meter in a JSON structure
	meterAsBytes, _ := json.Marshal(meter)

	//registers meter in the ledger
	stub.PutState(meterid, meterAsBytes)

	//loging...
	fmt.Println("Registering meter: ", meter)

	//notify procedure success
	return shim.Success(nil)
}

/*
This method implements the insertion of encrypted measurements in the blockchain.
The encryptation must uses the same public key configured to the meter.
Notice that the informed measurement will be added (accumulated) to the the previous
encrypted measurement consumption information.
The vector args[] must contain two parameters:
- args[0] - meter ID
- args[1] - the legally relevant information, in a string representing a big int number.
- args[2] - the signature digest, in base64 encode format.
*/
func (s *SmartContract) checkSignature(stub shim.ChaincodeStubInterface, args []string) sc.Response {

	//validate args vector lenght
	if len(args) != 3 {
		return shim.Error("It was expected 3 parameter: <meter ID> <information> <signature>")
	}

	//gets the parameter associated with the meter ID and the digital signature
	meterid := args[0]
	info := args[1]
	sign := args[2]

	//loging...
	fmt.Println("Testing args: ", meterid, info, sign)

	//retrive meter record
	meterAsBytes, err := stub.GetState(meterid)

	//test if we receive a valid meter ID
	if err != nil || meterAsBytes == nil {
		return shim.Error("Error on retrieving meter ID register")
	}

	//creates Meter struct to manipulate returned bytes
	MyMeter := Meter{}

	//loging...
	fmt.Println("Retrieving meter bytes: ", meterAsBytes)

	//convert bytes into a Meter object
	json.Unmarshal(meterAsBytes, &MyMeter)

	//decode de public key to the internal format
	pubkey := PublicKeyDecodePEM(MyMeter.PubKey)

	fmt.Println("pubkey: ", pubkey)

	//loging...
	fmt.Println("Retrieving meter after unmarshall: ", MyMeter)

	//calculates the information hash
	hash := sha256.Sum256([]byte(info))

	//now we decode the signature to extract the DER-encoded byte string
	der, err := base64.StdEncoding.DecodeString(sign)
	if err != nil {
		return shim.Error("Error on decode the digital signature")
	}

	//creates a signature data structure
	sig := &ECDSASignature{}

	//unmarshal the R and S components of the ASN.1-encoded signature
	_, err = asn1.Unmarshal(der, sig)
	if err != nil {
		return shim.Error("Error on get R and S terms from the digital signature")
	}

	//validates de digital signature
	valid := ecdsa.Verify(&pubkey, hash[:], sig.R, sig.S)

	// buffer is a JSON array containing records
	var buffer bytes.Buffer
	buffer.WriteString("[")
	buffer.WriteString("\"Counter\":")
	buffer.WriteString(strconv.FormatBool(valid))
	buffer.WriteString("]")

	//notify procedure success
	return shim.Success(buffer.Bytes())
}

/*
This method is a dummy test that makes the endorser "sleep" for some seconds.
It is usefull to check either the sleeptime affects the performance of concurrent
transactions.
- args[0] - sleeptime (in seconds)
*/
func (s *SmartContract) sleepTest(stub shim.ChaincodeStubInterface, args []string) sc.Response {
	//validate args vector lenght
	if len(args) != 1 {
		return shim.Error("It was expected 1 parameter: <sleeptime>")
	}

	//gets the parameter associated with the meter ID and the incremental measurement
	sleeptime, err := strconv.Atoi(args[0])

	//test if we receive a valid meter ID
	if err != nil {
		return shim.Error("Error on retrieving sleep time")
	}

	//tests if sleeptime is a valid value
	if sleeptime > 0 {
		//stops during sleeptime seconds
		time.Sleep(time.Duration(sleeptime) * time.Second)
	}

	//return payload with bytes related to the meter state
	return shim.Success(nil)
}

/*
This method brings the changing history of a specific meter asset. It can be useful to
query all the changes that happened with a meter value.
- args[0] - asset key (or meter ID)
*/
func (s *SmartContract) queryHistory(stub shim.ChaincodeStubInterface, args []string) sc.Response {

	//validate args vector lenght
	if len(args) != 1 {
		return shim.Error("It was expected 1 parameter: <key>")
	}

	historyIer, err := stub.GetHistoryForKey(args[0])

	//verifies if the history exists
	if err != nil {
		//fmt.Println(errMsg)
		return shim.Error("Fail on getting ledger history")
	}

	// buffer is a JSON array containing records
	var buffer bytes.Buffer
	var counter = 0
	buffer.WriteString("[")
	bArrayMemberAlreadyWritten := false
	for historyIer.HasNext() {
		//increments iterator
		queryResponse, err := historyIer.Next()
		if err != nil {
			return shim.Error(err.Error())
		}
		// Add a comma before array members, suppress it for the first array member
		if bArrayMemberAlreadyWritten == true {
			buffer.WriteString(",")
		}

		//generates a formated result
		buffer.WriteString("{\"Value\":")
		buffer.WriteString("\"")
		// Record is a JSON object, so we write as-is
		buffer.WriteString(string(queryResponse.Value))
		buffer.WriteString("\"")
		buffer.WriteString(", \"Counter\":")
		buffer.WriteString(strconv.Itoa(counter))
		//buffer.WriteString(queryResponse.Timestamp)
		buffer.WriteString("}")
		bArrayMemberAlreadyWritten = true

		//increases counter
		counter++
	}
	buffer.WriteString("]")
	historyIer.Close()

	//loging...
	fmt.Printf("Consulting ledger history, found %d\n records", counter)

	//notify procedure success
	return shim.Success(buffer.Bytes())
}

/*
This method brings the number of times that a meter asset was modified in the ledger.
It performs faster than queryHistory() method once it does not retrive any information,
it only counts the changes.
- args[0] - asset key (or meter ID)
*/
func (s *SmartContract) countHistory(stub shim.ChaincodeStubInterface, args []string) sc.Response {

	//validate args vector lenght
	if len(args) != 1 {
		return shim.Error("It was expected 1 parameter: <key>")
	}

	historyIer, err := stub.GetHistoryForKey(args[0])

	//verifies if the history exists
	if err != nil {
		//fmt.Println(errMsg)
		return shim.Error("Fail on getting ledger history")
	}

	//creates a counter
	var counter int64
	counter = 0

	for historyIer.HasNext() {
		//increments iterator
		_, err := historyIer.Next()
		if err != nil {
			return shim.Error(err.Error())
		}

		//increases counter
		counter++

		fmt.Printf("Consulting ledger history, found %d\n records", counter)
	}
	// buffer is a JSON array containing records
	var buffer bytes.Buffer
	buffer.WriteString("[")
	buffer.WriteString("\"Counter\":")
	buffer.WriteString(strconv.FormatInt(counter, 10))
	buffer.WriteString("]")

	historyIer.Close()

	//loging...
	fmt.Printf("Consulting ledger history, found %d\n records", counter)

	//notify procedure success
	return shim.Success(buffer.Bytes())
}

func (s *SmartContract) EnumerateHistory(stub shim.ChaincodeStubInterface, args []string) sc.Response {

	//validate args vector lenght
	if !(len(args) == 3) {
		return shim.Error("Eram esperado três parametros: Placa, Combustível e Distância")
	}

	//gets the parameters associated with the meter ID and the public key (in PEM format)
	placa := args[0] // chave
	combustivel := args[1]
	distancia := args[2]
	fmt.Println("Recebidos os dados: ", placa, " ", combustivel, " ", distancia)

	//creates the meter record with the respective public key
	var trajeto = Trajeto{Combustivel: combustivel, Distancia: distancia}

	//encapsulates meter in a JSON structure
	trajetoAsBytes, _ := json.Marshal(trajeto) //valor

	//acesso a interface do blockchain para gravar informações
	stub.PutState(placa, trajetoAsBytes)

	//loging...
	fmt.Println("Gravando o trajeto: ", trajeto)

	//notify procedure success
	return shim.Success(nil)
}

func (s *SmartContract) Mynum(stub shim.ChaincodeStubInterface, args []string) sc.Response {

	//validate args vector lenght
	if !(len(args) == 3) {
		return shim.Error("Eram esperado três parametros: Placa, Combustível e Distância")
	}

	// obter os parâmetros passados
	placa := args[0] //chave
	Mynum1 := args[1]
	Mynum2 := args[2]

	// converter os valores para inteiros

	combustivel, err := strconv.Atoi(Mynum1)
	if err != nil {
		return shim.Error("Erro ao converter o valor do combustível")
	}

	distancia, err := strconv.Atoi(Mynum2)
	if err != nil {
		return shim.Error("Erro ao converter o valor da distância")
	}

	// calcular a eficiência energética
	eficiencia := float64(distancia) / float64(combustivel)
	fmt.Println("Eficiência Energética:", eficiencia)

	fmt.Println("Recebidos os dados:", placa, combustivel, distancia)

	// criar o registro do trajeto com os respectivos valores
	trajeto := Trajeto{
		Combustivel: strconv.Itoa(combustivel),
		Distancia:   strconv.Itoa(distancia),
	}

	// converter o trajeto em uma estrutura JSON
	trajetoAsBytes, err := json.Marshal(trajeto)
	if err != nil {
		return shim.Error("Erro ao converter o trajeto em JSON")
	}

	// gravar o trajeto no estado do ledger
	err = stub.PutState(placa, trajetoAsBytes)
	if err != nil {
		return shim.Error("Erro ao gravar o trajeto no estado do ledger")
	}

	fmt.Println("Gravando o trajeto:", trajeto)

	// notificar o sucesso da execução
	return shim.Success(nil)
}

/*
This method counts the total of well succeeded transactions in the ledger.
*/
func (s *SmartContract) countLedger(stub shim.ChaincodeStubInterface) sc.Response {

	//use a range of keys, assuming that the max key value is 999999,
	resultsIterator, err := stub.GetStateByRange("0", "999999")
	if err != nil {
		return shim.Error(err.Error())
	}

	//defer iterator closes at the end of the function
	defer resultsIterator.Close()

	//creates a counter
	var counter int64
	var keys int64
	counter = 0
	keys = 0

	//the interator checks all the valid keys
	for resultsIterator.HasNext() {

		//increments iterator
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}

		//busca historico da proxima key
		historyIer, err := stub.GetHistoryForKey(queryResponse.Key)

		//verifies if the history exists
		if err != nil {
			//fmt.Println(errMsg)
			return shim.Error(err.Error())
		}

		defer historyIer.Close()

		for historyIer.HasNext() {
			//increments iterator
			_, err := historyIer.Next()
			if err != nil {
				return shim.Error(err.Error())
			}

			//increases counter
			counter++
		}
		fmt.Printf("Consulting ledger history, found key %s\n", queryResponse.Key)

		keys++
	}
	// buffer is a JSON array containing records
	var buffer bytes.Buffer
	buffer.WriteString("[")
	buffer.WriteString("\"Counter\":")
	buffer.WriteString(strconv.FormatInt(counter, 10))
	buffer.WriteString("\"Keys\":")
	buffer.WriteString(strconv.FormatInt(keys, 10))
	buffer.WriteString("]")

	//loging...
	fmt.Printf("Consulting ledger history, found %d transactions in %d keys\n", counter, keys)

	//notify procedure success
	return shim.Success(buffer.Bytes())
}

/*
This method executes a free query on the ledger, returning a vector of meter assets.
The query string must be a query expression supported by CouchDB servers.
- args[0] - query string.
*/
func (s *SmartContract) queryLedger(stub shim.ChaincodeStubInterface, args []string) sc.Response {

	//validate args vector lenght
	if len(args) != 1 {
		return shim.Error("It was expected 1 parameter: <query string>")
	}

	//using auxiliar variable
	queryString := args[0]

	//loging...
	fmt.Printf("Executing the following query: %s\n", queryString)

	//try to execute query and obtain records iterator
	resultsIterator, err := stub.GetQueryResult(queryString)
	//test if iterator is valid
	if err != nil {
		return shim.Error(err.Error())
	}
	//defer iterator closes at the end of the function
	defer resultsIterator.Close()

	// buffer is a JSON array containing QueryRecords
	var buffer bytes.Buffer
	buffer.WriteString("[")
	bArrayMemberAlreadyWritten := false
	for resultsIterator.HasNext() {
		//increments iterator
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}
		// Add a comma before array members, suppress it for the first array member
		if bArrayMemberAlreadyWritten == true {
			buffer.WriteString(",")
		}

		//generates a formated result
		buffer.WriteString("{\"Key\":")
		buffer.WriteString("\"")
		buffer.WriteString(queryResponse.Key)
		buffer.WriteString("\"")
		buffer.WriteString(", \"Record\":")
		// Record is a JSON object, so we write as-is
		buffer.WriteString(string(queryResponse.Value))
		buffer.WriteString("}")
		bArrayMemberAlreadyWritten = true
	}
	buffer.WriteString("]")

	//loging...
	fmt.Printf("Obtained the following fill up records: %s\n", buffer.String())

	//notify procedure success
	return shim.Success(buffer.Bytes())
}

func (s *SmartContract) registerWeatherFromWeb(stub shim.ChaincodeStubInterface, args []string) sc.Response {

	//validate args vector lenght
	if len(args) != 3 {
		return shim.Error("It was expected the parameters: <\"city name\"> <situation> <temperature>")
	}

	//gets the parameters
	cityName := args[0]
	situation := args[1]
	temperature := args[2]

	// Receives the time of creation
	timestamp := time.Now() // 	// 2009-11-10 23:00:00 +0000 UTC m=+0.000000001
	//	timestampString := timestamp.String() 

	// extract date
	year, month, day := timestamp.Date()
	dateString := fmt.Sprintf("%d-%02d-%02d", year, month, day)

	// extract hour
	hour, minute, second := timestamp.Clock()
	hourString := fmt.Sprintf("%02d:%02d:%02d", hour, minute, second)

	//creates the meter record with the respective public key
	// var station = Station{PubKey: strpubkey, MyDate: creationDate}
	var weatherApi = WeatherAPI{CityName: cityName, Situation: situation, Temperature: temperature, Date: dateString, Hour: hourString}

	//encapsulates station data in a JSON structure
	weatherApiAsBytes, _ := json.Marshal(weatherApi)

	//loging...
	fmt.Println("Registering climate info from web...")

	//registers meter in the ledger
	stub.PutState(cityName, weatherApiAsBytes)

	//notify procedure success
	return shim.Success(nil)
}

func (s *SmartContract) getWeatherFromWeb(stub shim.ChaincodeStubInterface, args []string) sc.Response {

	//validate args vector lenght
	if len(args) != 1 {
		return shim.Error("It was expected the parameters: <\"city name\">")
	}

	//gets the parameters
	cityName := args[0]
	fmt.Println(cityName)

	// retrieve the station data from the ledger
	weatherApiAsBytes, err := stub.GetState(cityName)
	if err != nil {
		fmt.Println(err)
		return shim.Error("Error retrieving station from the ledger")
	}

	// check if its null
	if weatherApiAsBytes == nil {
		return shim.Error("No info registered for this city")
	}

	//creates Station struct to manipulate returned bytes
	MyWeather := WeatherAPI{}

	//loging...
	fmt.Println("Retrieving station data: ", weatherApiAsBytes)

	//convert bytes into a station object
	json.Unmarshal(weatherApiAsBytes, &MyWeather)

	// log
	fmt.Println("Retrieving station data after unmarshall: ", MyWeather)

	cityName = string(MyWeather.CityName)
	situation := string(MyWeather.Situation)
	temperature := string(MyWeather.Temperature)
	date := string(MyWeather.Date)
	hour := string(MyWeather.Hour)

	var info = "Cityname: " + cityName +
		"\nSituation: " + situation +
		"\nTemperature: " + temperature +
		"\nTimestamp: " + date + " " + hour

	// returns all station info
	return shim.Success(
		[]byte(info),
	)
}

func (s *SmartContract) queryWebWeatherHistory(stub shim.ChaincodeStubInterface, args []string) sc.Response {
	
	if len(args) != 2 {
		return shim.Error("It was expected 2 parameters: <key> <year-month-day>")
	}

	historyIer, err := stub.GetHistoryForKey(args[0])
	wantedDate := args[1]

	//verifies if the history exists
	if err != nil {
		//fmt.Println(errMsg)
		return shim.Error("Fail on getting ledger history")
	}

	// flag de achou ou nao
	flag := 0
	errMsg := ""

	for historyIer.HasNext() {
		queryResponse, err := historyIer.Next()
		if err != nil {
			return shim.Error(err.Error())
		}

		/*
			- Pegar queryResponse.value da chave CityName (virá em Bytes. Separar em uma lista JSON)
			- Separar Timestamp (formatado ou separado?) e comparar com o inserido
			- Se coincidirem, retornar clima
			- Se não, ir para o próximo
			- Caso nenhum seja encontrado, retorne o aviso
		*/

		valorBytes := queryResponse.Value
		_ = valorBytes

		MyWeather := WeatherAPI{}

		json.Unmarshal(valorBytes, &MyWeather)
		fmt.Println("Retrieving station data after unmarshall: ", MyWeather)

		date := string(MyWeather.Date)

		if wantedDate == date {
			cityName := string(MyWeather.CityName) // talvez nem precise
			situation := string(MyWeather.Situation)
			temperature := string(MyWeather.Temperature)
			date := string(MyWeather.Date)
			hour := string(MyWeather.Hour)

			var info = "Cityname: " + cityName +
				"\nSituation: " + situation +
				"\nTemperature: " + temperature +
				"\nTimestamp: " + date + " " + hour

			flag++
			return shim.Success([]byte(info))
		}

	}
	historyIer.Close()

	//loging...
	// fmt.Printf("Consulting ledger history, found %d\n records", counter)

	if flag == 0 {
		// const shimErr = shim.Error("Não encontrado")
		errMsg = "Não encontrado"
	}

	return shim.Error(errMsg)
}

/*
 * The main function starts up the chaincode in the container during instantiate
 */
func main() {

	////////////////////////////////////////////////////////
	// USE THIS BLOCK TO COMPILE THE CHAINCODE
	if err := shim.Start(new(SmartContract)); err != nil {
		fmt.Printf("Error starting SmartContract chaincode: %s\n", err)
	}
	////////////////////////////////////////////////////////

	////////////////////////////////////////////////////////
	// USE THIS BLOCK TO PERFORM ANY TEST WITH THE CHAINCODE

	// //create pair of keys
	// privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	// if err != nil {
	// 	panic(err)
	// }

	// //marshal the keys in a buffer
	// e, err := json.Marshal(privateKey)
	// if err != nil {
	// 	fmt.Println(err)
	// 	return
	// }

	// _ = ioutil.WriteFile("ecdsa-keys.json", e, 0644)

	// //read the saved key
	// file, _ := ioutil.ReadFile("ecdsa-keys.json")

	// myPrivKey := ecdsa.PrivateKey{}
	// //myPubKey := ecdsa.PublicKey{}

	// _ = json.Unmarshal([]byte(file), &myPrivKey)

	// fmt.Println("Essa é minha chave privada:")
	// fmt.Println(myPrivKey)

	// myPubKey := myPrivKey.PublicKey

	// //test digital signature verifying
	// msg := "message"
	// hash := sha256.Sum256([]byte(msg))
	// fmt.Println("hash: ", hash)

	// r, s, err := ecdsa.Sign(rand.Reader, privateKey, hash[:])
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Printf("signature: (0x%x, 0x%x)\n", r, s)

	// myPubKey.Curve = elliptic.P256()

	// fmt.Println("Essa é minha chave publica:")
	// fmt.Println(myPubKey)

	// valid := ecdsa.Verify(&myPubKey, hash[:], r, s)
	// fmt.Println("signature verified:", valid)

	// otherpk := "-----BEGIN PUBLIC KEY-----\nMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE6NXETwtkAKGWBcIsI6/OYE0EwsVj\n3Fc4hHTaReNfq6Hz2UEzsJKCYN0stjPCXbpdUlYtETC1a3EcS3SUVYX6qA==\n-----END PUBLIC KEY-----\n"

	// newkey := PublicKeyDecodePEM(otherpk)
	// myPubKey.Curve = elliptic.P256()

	// //valid = ecdsa.Verify(newkey, hash[:], r, s)
	// //fmt.Println("signature verified:", valid)

	// mysign := "MEYCIQCY16jbdY222oEpFiSRwXPi1kS7c4wuwxYXeWJOoAjnVgIhAJQTM+itbm1mQyd40Ug0xr2/AvjZmFSdoc/iSSHA6nRI"

	// // first decode the signature to extract the DER-encoded byte string
	// der, err := base64.StdEncoding.DecodeString(mysign)
	// if err != nil {
	// 	panic(err)
	// }

	// // unmarshal the R and S components of the ASN.1-encoded signature into our
	// // signature data structure
	// sig := &ECDSASignature{}
	// _, err = asn1.Unmarshal(der, sig)
	// if err != nil {
	// 	panic(err)
	// }

	// valid = ecdsa.Verify(&newkey, hash[:], sig.R, sig.S)
	// fmt.Println("signature verified:", valid)

	// fmt.Println("Curve: ", newkey.Curve.Params())

	////////////////////////////////////////////////////////

}
