package ro.pub.cs.systems.ssproject.mqtt
 
object MqttConstants {
    const val TAG = "MqttHandler"
    const val TOPIC_IMAGES = "ssproject/images"
    const val TOPIC_COMMANDS = "ssproject/commands"
 
    const val CMD_CAPTURE = "CAPTURE"
    const val CMD_START_LIVE = "START-LIVE"
    const val CMD_STOP_LIVE = "STOP-LIVE"
 
    // TLS
    const val BKS_STORE_TYPE = "BKS"
    const val TLS_PROTOCOL = "TLSv1.2"
    const val KEYSTORE_PASSWORD = "changeit"
    const val TRUSTSTORE_PASSWORD = "changeit"
    const val DEFAULT_TLS_PORT = "8883"
}