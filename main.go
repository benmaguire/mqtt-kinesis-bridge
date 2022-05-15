package main

import (
        "os"
        "os/signal"
        "syscall"
        "crypto/tls"
        "crypto/x509"
        "io/ioutil"
        log "github.com/sirupsen/logrus"

        MQTT "github.com/eclipse/paho.mqtt.golang"

        "github.com/aws/aws-sdk-go/aws"
        "github.com/aws/aws-sdk-go/aws/session"
        "github.com/aws/aws-sdk-go/service/firehose"
)



var streamName string
var sess = session.Must(session.NewSession())
var firehoseService = firehose.New(sess, aws.NewConfig().WithRegion("ap-southeast-2"))


var f MQTT.MessageHandler = func(client MQTT.Client, msg MQTT.Message) {
        log.Debug("MQTT MSG: " + string(msg.Payload()))

	// Put to Kinesis
        var rec firehose.Record
        var recInput firehose.PutRecordInput

        rec.SetData(msg.Payload())
        recInput.SetDeliveryStreamName(streamName)
        recInput.SetRecord(&rec)
        res, err1 := firehoseService.PutRecord(&recInput)

        if err1 != nil {
                log.Fatal(err1)
        }
        log.Debug(res)
}



func NewTLSConfig() *tls.Config {
        // Import trusted certificates from CAfile.pem.
        // Alternatively, manually add CA certificates to
        // default openssl CA bundle.
        certpool := x509.NewCertPool()
        pemCerts, err := ioutil.ReadFile("samplecerts/CAfile.pem")
        if err == nil {
                certpool.AppendCertsFromPEM(pemCerts)
        }

        // Create tls.Config with desired tls properties
        return &tls.Config{
                // RootCAs = certs used to verify server cert.
                RootCAs: certpool,
                // ClientAuth = whether to request cert from server.
                // Since the server is set up for SSL, this happens
                // anyways.
                ClientAuth: tls.NoClientCert,
                // ClientCAs = certs used to validate client cert.
                ClientCAs: nil,
                // InsecureSkipVerify = verify that cert contents
                // match server. IP matches what is in cert etc.
                InsecureSkipVerify: true,
        }
}



func main() {

        log.Info("Starting Application")

	// Get Debug from EV
        logdebug := os.Getenv("LOG_DEBUG")
        if logdebug == "true" {
                log.SetLevel(log.DebugLevel)
                log.Info("Setting Log Debug On")
        }


	// Create Channel for Wait
        c := make(chan os.Signal, 1)
        signal.Notify(c, os.Interrupt, syscall.SIGTERM)


        // Get Environment Vars
        broker := os.Getenv("MQTT_BROKER")
        clientid := os.Getenv("MQTT_CLIENTID")
        topic := os.Getenv("MQTT_TOPIC")
        username := os.Getenv("MQTT_USER")
        password := os.Getenv("MQTT_PASS")
        streamName = os.Getenv("FIREHOSE_STREAM")


        // MQTT
	tlsconfig := NewTLSConfig()
        opts := MQTT.NewClientOptions()
        opts.AddBroker(broker)
        opts.SetClientID(clientid).SetTLSConfig(tlsconfig)
        opts.SetUsername(username)
        opts.SetPassword(password)

        opts.OnConnect = func(c MQTT.Client) {
            if token := c.Subscribe(topic, 0, f); token.Wait() && token.Error() != nil {
                    panic(token.Error())
            }
        }

        // Start the MQTT connection
        log.Info("Connecting to MQTT")
        client := MQTT.NewClient(opts)
        if token := client.Connect(); token.Wait() && token.Error() != nil {
            panic(token.Error())
        } else {
            log.Info("Connected to MQTT Server\n")
        }


	// Wait for Notify
        <-c

        log.Info("Disconnecting from MQTT")
        client.Disconnect(250)
}


