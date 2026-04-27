package ro.pub.cs.systems.ssproject.mqtt

object MqttConstants {
    const val TAG = "MqttHandler"
    const val TOPIC_IMAGES = "ssproject/images"
    const val TOPIC_COMMANDS = "ssproject/commands"

    const val CMD_CAPTURE = "CAPTURE"
    const val CMD_START_LIVE = "START-LIVE"
    const val CMD_STOP_LIVE = "STOP-LIVE"
}