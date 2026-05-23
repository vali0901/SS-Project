package ro.pub.cs.systems.ssproject.ui.setup

import android.content.Intent
import android.os.Bundle
import android.util.Patterns
import android.widget.Button
import android.widget.EditText
import android.widget.Toast
import androidx.activity.ComponentActivity
import androidx.activity.enableEdgeToEdge
import androidx.lifecycle.lifecycleScope
import com.google.android.material.materialswitch.MaterialSwitch
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import org.json.JSONObject
import ro.pub.cs.systems.ssproject.mqtt.MqttConstants
import ro.pub.cs.systems.ssproject.ui.dashboard.MainActivity
import ro.pub.cs.systems.ssproject.R
import java.io.OutputStreamWriter
import java.net.HttpURLConnection
import java.net.URL

class SetupActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        setContentView(R.layout.activity_setup)

        val ipField = findViewById<EditText>(R.id.setup_card_view_ip_field)
        val portField = findViewById<EditText>(R.id.setup_card_view_port_field)
        val emailField = findViewById<EditText>(R.id.setup_card_view_email_field)
        val passwordField = findViewById<EditText>(R.id.setup_card_view_password_field)
        val tlsSwitch = findViewById<MaterialSwitch>(R.id.setup_card_view_tls_switch)
        val connect = findViewById<Button>(R.id.setup_card_view_connect_btn)

        // Actualizarea portului implicit la activarea TLS
        tlsSwitch.setOnCheckedChangeListener { _, isChecked ->
            if (isChecked && portField.text.toString() == "1883") {
                portField.setText(MqttConstants.DEFAULT_TLS_PORT)
            } else if (!isChecked && portField.text.toString() ==
                MqttConstants.DEFAULT_TLS_PORT) {
                portField.setText("1883")
            }
        }

        connect.setOnClickListener {
            val inputIp = ipField.text.toString().trim()
            val inputPort = portField.text.toString().trim()
            val email = emailField.text.toString().trim()
            val password = passwordField.text.toString()

            if (inputIp.isEmpty() || !Patterns.IP_ADDRESS.matcher(inputIp).matches()) {
                ipField.error = "Invalid IP address"
                ipField.requestFocus()
                return@setOnClickListener
            }

            val portNumber = inputPort.toIntOrNull()
            if (portNumber == null || portNumber !in 1..65535) {
                portField.error = "Invalid port"
                portField.requestFocus()
                return@setOnClickListener
            }

            if (portNumber == MqttConstants.DEFAULT_TLS_PORT.toInt() && !tlsSwitch.isChecked) {
                portField.error = "Port 8883 requires mTLS"
                tlsSwitch.isChecked = true
                portField.requestFocus()
                return@setOnClickListener
            }

            if (tlsSwitch.isChecked && portNumber != MqttConstants.DEFAULT_TLS_PORT.toInt()) {
                portField.error = "Use port ${MqttConstants.DEFAULT_TLS_PORT} for mTLS"
                portField.requestFocus()
                return@setOnClickListener
            }

            if (email.isEmpty() || !Patterns.EMAIL_ADDRESS.matcher(email).matches()) {
                emailField.error = "Invalid email"
                emailField.requestFocus()
                return@setOnClickListener
            }

            if (password.isEmpty()) {
                passwordField.error = "Password is required"
                passwordField.requestFocus()
                return@setOnClickListener
            }

            connect.isEnabled = false
            lifecycleScope.launch {
                val token = login(inputIp, email, password)
                connect.isEnabled = true

                if (token == null) {
                    Toast.makeText(this@SetupActivity, "Login failed", Toast.LENGTH_LONG).show()
                    return@launch
                }

                val intent = Intent(this@SetupActivity, MainActivity::class.java)
                intent.putExtra("brokerIp", inputIp)
                intent.putExtra("brokerPort", inputPort)
                intent.putExtra("useTls", tlsSwitch.isChecked)
                intent.putExtra("authToken", token)
                startActivity(intent)
            }
        }
    }

    private suspend fun login(serverIp: String, email: String, password: String): String? {
        return withContext(Dispatchers.IO) {
            val connection = (URL("http://$serverIp:8080/login").openConnection() as HttpURLConnection).apply {
                requestMethod = "POST"
                connectTimeout = 10000
                readTimeout = 10000
                setRequestProperty("Content-Type", "application/json")
                doOutput = true
            }

            try {
                val payload = JSONObject()
                    .put("email", email)
                    .put("password", password)
                    .toString()

                OutputStreamWriter(connection.outputStream, Charsets.UTF_8).use {
                    it.write(payload)
                }

                if (connection.responseCode != HttpURLConnection.HTTP_OK) {
                    return@withContext null
                }

                val body = connection.inputStream.bufferedReader().use { it.readText() }
                JSONObject(body).optString("token").takeIf { it.isNotBlank() }
            } catch (_: Exception) {
                null
            } finally {
                connection.disconnect()
            }
        }
    }
}
