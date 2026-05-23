package ro.pub.cs.systems.ssproject.mqtt

import android.os.Build
import android.util.Log
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import org.json.JSONObject
import org.eclipse.paho.client.mqttv3.IMqttDeliveryToken
import org.eclipse.paho.client.mqttv3.MqttCallback
import org.eclipse.paho.client.mqttv3.MqttClient
import org.eclipse.paho.client.mqttv3.MqttConnectOptions
import org.eclipse.paho.client.mqttv3.MqttMessage
import org.eclipse.paho.client.mqttv3.persist.MemoryPersistence
import java.net.NetworkInterface
import java.util.Locale
import javax.net.ssl.SSLSocketFactory

class MqttHandler(
    brokerIp: String,
    private val brokerPort: String,
    private val authToken: String,
    private val sslSocketFactory: SSLSocketFactory? = null,
    private val isConnectedCallback: (Boolean) -> Unit,
    private val onCommandReceived: (String) -> Unit
) : MqttCallback {
    // Determinăm protocolul pe baza prezenței SSLSocketFactory
    private val protocol = if (sslSocketFactory != null) "ssl" else "tcp"
    private val brokerUrl = "$protocol://$brokerIp:$brokerPort"
    private val deviceId = buildDeviceId()
    private val deviceName = "${Build.MANUFACTURER} ${Build.MODEL}".trim()
    private val clientId = "${deviceId}_${MqttClient.generateClientId()}"
    private val registerTopic = "${MqttConstants.TOPIC_REGISTER}/$deviceId"
    private val imageTopic = "${MqttConstants.TOPIC_IMAGES}/$deviceId"
    private var client: MqttClient? = null

    suspend fun connect() {
        withContext(Dispatchers.IO) {
            if (client?.isConnected == true) {
                return@withContext
            }

            if (brokerPort == MqttConstants.DEFAULT_TLS_PORT && sslSocketFactory == null) {
                Log.e(MqttConstants.TAG, "TLS port ${MqttConstants.DEFAULT_TLS_PORT} requires an SSL socket factory")
                isConnectedCallback(false)
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

                registerDevice()

                client?.subscribe(MqttConstants.TOPIC_COMMANDS, 1)
                Log.i(MqttConstants.TAG, "Subscribed to ${MqttConstants.TOPIC_COMMANDS}")
            } catch (e: Exception) {
                Log.e(MqttConstants.TAG, "Connection error: ${e.message}")
                e.printStackTrace()
                try {
                    client?.close()
                } catch (_: Exception) {}
                client = null
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
-0
    suspend fun publishImage(imageBytes: ByteArray, qos: Int = 0): Boolean {
        return withContext(Dispatchers.IO) {
            if (!isConnected()) {
                Log.w(MqttConstants.TAG, "Cannot publish: Client is not connected")
                return@withContext false
            }

            try {
                val message = MqttMessage(imageBytes)
                message.qos = qos
                message.isRetained = false

                client?.publish(imageTopic, message)
                true
            } catch (e: Exception) {
                Log.e(MqttConstants.TAG, "Publish error: ${e.message}")
                false
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

    private fun registerDevice() {
        val payload = JSONObject()
            .put("name", deviceName)
            .put("ip", getLocalIpAddress())
            .put("port", brokerUrl.substringAfterLast(":"))
            .put("token", authToken)
            .toString()

        val message = MqttMessage(payload.toByteArray(Charsets.UTF_8)).apply {
            qos = 1
            isRetained = false
        }

        client?.publish(registerTopic, message)
        Log.i(MqttConstants.TAG, "Registered device on $registerTopic")
    }

    private fun buildDeviceId(): String {
        val rawId = "android-${Build.MANUFACTURER}-${Build.MODEL}"
        return rawId
            .lowercase(Locale.ROOT)
            .replace(Regex("[^a-z0-9_-]+"), "-")
            .trim('-')
            .ifEmpty { "android-device" }
    }

    private fun getLocalIpAddress(): String {
        return try {
            NetworkInterface.getNetworkInterfaces().toList()
                .flatMap { it.inetAddresses.toList() }
                .firstOrNull { !it.isLoopbackAddress && it.hostAddress?.contains(":") == false }
                ?.hostAddress ?: "unknown"
        } catch (e: Exception) {
            Log.w(MqttConstants.TAG, "Could not resolve local IP: ${e.message}")
            "unknown"
        }
    }
}
