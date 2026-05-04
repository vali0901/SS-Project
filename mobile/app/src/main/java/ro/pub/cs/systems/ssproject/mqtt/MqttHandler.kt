package ro.pub.cs.systems.ssproject.mqtt

import android.os.Build
import android.util.Log
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import org.eclipse.paho.client.mqttv3.IMqttDeliveryToken
import org.eclipse.paho.client.mqttv3.MqttCallback
import org.eclipse.paho.client.mqttv3.MqttClient
import org.eclipse.paho.client.mqttv3.MqttConnectOptions
import org.eclipse.paho.client.mqttv3.MqttMessage
import org.eclipse.paho.client.mqttv3.persist.MemoryPersistence
import javax.net.ssl.SSLSocketFactory

class MqttHandler(
    brokerIp: String,
    brokerPort: String,
    private val sslSocketFactory: SSLSocketFactory? = null,
    private val isConnectedCallback: (Boolean) -> Unit,
    private val onCommandReceived: (String) -> Unit
) : MqttCallback {
    // Determinăm protocolul pe baza prezenței SSLSocketFactory
    private val protocol = if (sslSocketFactory != null) "ssl" else "tcp"
    private val brokerUrl = "$protocol://$brokerIp:$brokerPort"
    private val clientId = "${Build.MANUFACTURER}_${Build.MODEL}_${MqttClient.generateClientId()}"
    private var client: MqttClient? = null

    suspend fun connect() {
        withContext(Dispatchers.IO) {
            if (client?.isConnected == true) {
                return@withContext
            }

            try {
                client = MqttClient(brokerUrl, clientId, MemoryPersistence())
                client?.setCallback(this@MqttHandler)

                val options = MqttConnectOptions().apply {
                    isCleanSession = true
                    connectionTimeout = 10
                    keepAliveInterval = 60

                    // Configurarea TLS dacă SSLSocketFactory este disponibil
                    if (sslSocketFactory != null) {
                        socketFactory = sslSocketFactory
                    }
                }

                client?.connect(options)
                Log.i(MqttConstants.TAG, "Connected to $brokerUrl")

                client?.subscribe(MqttConstants.TOPIC_COMMANDS, 1)
                Log.i(MqttConstants.TAG, "Subscribed to ${MqttConstants.TOPIC_COMMANDS}")
            } catch (e: Exception) {
                Log.e(MqttConstants.TAG, "Connection error: ${e.message}")
                e.printStackTrace()
            } finally {
                isConnectedCallback(isConnected())
            }
        }
    }

    suspend fun disconnect() {
        withContext(Dispatchers.IO) {
            try {
                if (client?.isConnected == true) {
                    client?.disconnect()
                }
                client?.close()
                client = null
                Log.i(MqttConstants.TAG, "Disconnected")
                isConnectedCallback(false)
            } catch (e: Exception) {
                Log.e(MqttConstants.TAG, "Connection error: ${e.message}")
                e.printStackTrace()
            }
        }
    }

    fun isConnected(): Boolean {
        return client?.isConnected == true
    }

    suspend fun publishImage(imageBytes: ByteArray, qos: Int = 0) {
        withContext(Dispatchers.IO) {
            if (isConnected()) {
                try {
                    val message = MqttMessage(imageBytes)
                    message.qos = qos
                    message.isRetained = false

                    client?.publish(MqttConstants.TOPIC_IMAGES, message)
                } catch (e: Exception) {
                    Log.e(MqttConstants.TAG, "Publish error: ${e.message}")
                }
            } else {
                Log.w(MqttConstants.TAG, "Cannot publish: Client is not connected")
            }
        }
    }

    override fun connectionLost(cause: Throwable?) {
        Log.w(MqttConstants.TAG, "Connection lost: ${cause?.message}")
        isConnectedCallback(false)
    }

    override fun messageArrived(topic: String?, message: MqttMessage?) {
        if (topic == MqttConstants.TOPIC_COMMANDS) {
            val command = message?.toString() ?: return
            Log.i(MqttConstants.TAG, "Received command: $command")

            onCommandReceived(command)
        }
    }

    override fun deliveryComplete(token: IMqttDeliveryToken?) {}
}