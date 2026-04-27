package ro.pub.cs.systems.ssproject.ui.setup

import android.content.Intent
import android.os.Bundle
import android.util.Patterns
import android.widget.Button
import android.widget.EditText
import androidx.activity.ComponentActivity
import androidx.activity.enableEdgeToEdge
import ro.pub.cs.systems.ssproject.ui.dashboard.MainActivity
import ro.pub.cs.systems.ssproject.R

class SetupActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        setContentView(R.layout.activity_setup)

        val ipField = findViewById<EditText>(R.id.setup_card_view_ip_field)
        val portField = findViewById<EditText>(R.id.setup_card_view_port_field)
        val connect = findViewById<Button>(R.id.setup_card_view_connect_btn)
        connect.setOnClickListener {
            val inputIp = ipField.text.toString().trim()
            val inputPort = portField.text.toString().trim()

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

            val intent = Intent(this, MainActivity::class.java)
            intent.putExtra("brokerIp", inputIp)
            intent.putExtra("brokerPort", inputPort)
            startActivity(intent)
        }
    }
}