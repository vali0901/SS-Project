package ro.pub.cs.systems.ssproject.ui.dashboard

import android.Manifest.permission.CAMERA
import android.graphics.ImageFormat
import android.graphics.Rect
import android.graphics.YuvImage
import android.os.Bundle
import android.util.Log
import android.view.View
import androidx.activity.enableEdgeToEdge
import androidx.appcompat.app.AppCompatActivity
import androidx.camera.core.CameraSelector
import androidx.camera.core.ImageAnalysis
import androidx.camera.core.ImageProxy
import androidx.camera.core.Preview
import androidx.camera.lifecycle.ProcessCameraProvider
import androidx.camera.view.PreviewView
import androidx.core.content.ContextCompat
import androidx.lifecycle.ProcessLifecycleOwner
import androidx.lifecycle.lifecycleScope
import com.google.android.material.button.MaterialButton
import com.google.android.material.loadingindicator.LoadingIndicator
import com.google.android.material.materialswitch.MaterialSwitch
import com.google.android.material.slider.Slider
import com.google.android.material.textfield.TextInputEditText
import com.google.android.material.textfield.TextInputLayout
import com.google.android.material.textview.MaterialTextView
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.asExecutor
import kotlinx.coroutines.launch
import ro.pub.cs.systems.ssproject.R
import ro.pub.cs.systems.ssproject.mqtt.MqttConstants
import ro.pub.cs.systems.ssproject.mqtt.MqttHandler
import ro.pub.cs.systems.ssproject.utils.ImageUtils
import ro.pub.cs.systems.ssproject.utils.PermissionHandler
import java.io.ByteArrayOutputStream
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale
import java.util.concurrent.Executor

class MainActivity : AppCompatActivity() {
    // UI Components
    private lateinit var statusIndicator: View
    private lateinit var statusBrokerAddress: MaterialTextView
    private lateinit var statusConnection: MaterialTextView
    private lateinit var cameraPreviewView: PreviewView
    private lateinit var cameraLoadingIndicator: LoadingIndicator
    private lateinit var logsTextView: MaterialTextView

    // Controls
    private lateinit var liveSwitch: MaterialSwitch
    private lateinit var captureButton: MaterialButton
    private lateinit var autoSendSwitch: MaterialSwitch
    private lateinit var autoSendInputLayout: TextInputLayout
    private lateinit var autoSendInput: TextInputEditText
    private lateinit var qualitySlider: Slider

    // Helpers
    private var mqttHandler: MqttHandler? = null
    private lateinit var cameraPermissionHandler: PermissionHandler
    private lateinit var cameraExecutor: Executor

    // State variables
    @Volatile private var isConnecting = false
    @Volatile private var isLiveMode = false
    @Volatile private var isAutoSending = false
    @Volatile private var shouldCaptureImage = false
    @Volatile private var currentQuality = 90
    @Volatile private var autoSendIntervalMillis = 10000L

    // Timer helpers
    private var lastSentTime = 0L

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        setContentView(R.layout.activity_main)

        initializeViews()
        setupListeners()
        updateUiState()

        val brokerIp = intent.getStringExtra("brokerIp")!!
        val brokerPort = intent.getStringExtra("brokerPort")!!
        statusBrokerAddress.text = getString(R.string.main_status_broker_address_format, brokerIp, brokerPort)

        mqttHandler = MqttHandler(
            brokerIp,
            brokerPort,
            isConnectedCallback = { isConnected ->
                runOnUiThread {
                    handleConnectionStateChange(isConnected)
                }
            },
            onCommandReceived = { command ->
                runOnUiThread {
                    appendLog(getString(R.string.main_logs_command_received_entry, command))
                    handleReceivedCommand(command)
                }
            }
        )
        connectMqtt()

        cameraExecutor = Dispatchers.Default.asExecutor()
        cameraPermissionHandler = PermissionHandler(
            activity = this,
            permissions = arrayOf(CAMERA),
            rationaleTitle = R.string.permission_camera_rationale_title,
            rationaleDescription = R.string.permission_camera_rationale_description,
            settingsRedirectTitle = R.string.permission_camera_denied_title,
            settingsRedirectDescription = R.string.permission_camera_denied_description,
            onPermissionsGranted = {
                startCamera()
                cameraLoadingIndicator.visibility = View.GONE
            },
            onPermissionsDenied = { finish() }
        )
        cameraPermissionHandler.requestPermissions()
    }

    override fun onStart() {
        super.onStart()
        if (mqttHandler?.isConnected() == false) {
            connectMqtt()
        }
    }

    private fun initializeViews() {
        statusIndicator = findViewById(R.id.main_card_status_indicator)
        statusBrokerAddress = findViewById(R.id.main_card_status_broker_address)
        statusConnection = findViewById(R.id.main_card_status_connection_state)
        cameraPreviewView = findViewById(R.id.main_card_camera_preview_view)
        cameraLoadingIndicator = findViewById(R.id.main_card_camera_loading_indicator)
        logsTextView = findViewById(R.id.main_card_logs_no_scroll)

        liveSwitch = findViewById(R.id.main_card_live_switch)
        captureButton = findViewById(R.id.main_controls_layout_capture)
        autoSendSwitch = findViewById(R.id.main_controls_layout_auto_send_switch)
        autoSendInputLayout = findViewById(R.id.main_controls_layout_auto_send_text_layout)
        autoSendInput = findViewById(R.id.main_controls_layout_auto_send_text)
        qualitySlider = findViewById(R.id.main_controls_layout_image_quality)
    }

    private fun setupListeners() {
        qualitySlider.addOnChangeListener { _, value, _ ->
            currentQuality = value.toInt()
        }

        captureButton.setOnClickListener {
            shouldCaptureImage = true
        }

        liveSwitch.setOnCheckedChangeListener { _, isChecked ->
            isLiveMode = isChecked
            if (isChecked) {
                appendLog(getString(R.string.main_logs_live_mode_on_entry))
            } else {
                appendLog(getString(R.string.main_logs_live_mode_off_entry))
            }
            updateUiState()
        }

        autoSendSwitch.setOnCheckedChangeListener { _, isChecked ->
            isAutoSending = isChecked

            if (isChecked) {
                val sec = autoSendInput.text.toString().toLongOrNull() ?: 5
                autoSendIntervalMillis = if (sec < 1) 1000L else sec * 1000L

                appendLog(getString(R.string.main_logs_auto_send_on_entry, autoSendIntervalMillis / 1000))
            } else {
                appendLog(getString(R.string.main_logs_auto_send_off_entry))
            }
            updateUiState()
        }
    }

    private fun updateUiState() {
        val isConnected = mqttHandler?.isConnected() == true

        if (!isConnected) {
            captureButton.isEnabled = false
            autoSendSwitch.isEnabled = false
            autoSendInputLayout.isEnabled = false
            autoSendInput.isEnabled = false
            liveSwitch.isEnabled = false
            qualitySlider.isEnabled = false
            return
        }

        qualitySlider.isEnabled = true

        if (isLiveMode) {
            captureButton.isEnabled = false
            autoSendSwitch.isEnabled = false
            autoSendInputLayout.isEnabled = false
            autoSendInput.isEnabled = false
            liveSwitch.isEnabled = true
        } else {
            autoSendSwitch.isEnabled = true
            liveSwitch.isEnabled = true

            if (isAutoSending) {
                captureButton.isEnabled = false
                autoSendInputLayout.isEnabled = false
                autoSendInput.isEnabled = false
            } else {
                captureButton.isEnabled = true
                autoSendInputLayout.isEnabled = true
                autoSendInput.isEnabled = true
            }
        }
    }

    private fun connectMqtt() {
        if (isConnecting) {
            return
        }
        isConnecting = true
        lifecycleScope.launch {
            mqttHandler?.connect()
            isConnecting = false
        }
        updateUiState()
    }

    private fun handleReceivedCommand(command: String) {
        when (command) {
            MqttConstants.CMD_CAPTURE -> {
                if (!isLiveMode) {
                    shouldCaptureImage = true
                } else {
                    appendLog(getString(R.string.main_logs_command_capture_ignored_entry))
                }
            }
            MqttConstants.CMD_START_LIVE -> {
                liveSwitch.isChecked = true
            }
            MqttConstants.CMD_STOP_LIVE -> {
                liveSwitch.isChecked = false
            }
            else -> appendLog(getString(R.string.main_logs_command_unknown_entry))
        }
    }

    private fun handleConnectionStateChange(isConnected: Boolean) {
        if (isConnected) {
            statusConnection.text = getString(R.string.main_status_connected)
            statusIndicator.setBackgroundResource(R.drawable.circle_green_12)
        } else {
            statusConnection.text = getString(R.string.main_status_disconnected)
            statusIndicator.setBackgroundResource(R.drawable.circle_red_12)
        }

        updateUiState()
    }

    private fun startCamera() {
        val cameraProviderFuture = ProcessCameraProvider.getInstance(this)

        cameraProviderFuture.addListener({
            val cameraProvider = cameraProviderFuture.get()

            val preview = Preview.Builder().build()
            preview.surfaceProvider = cameraPreviewView.surfaceProvider

            val imageAnalysis = ImageAnalysis.Builder()
                .setBackpressureStrategy(ImageAnalysis.STRATEGY_KEEP_ONLY_LATEST)
                .build()

            imageAnalysis.setAnalyzer(ContextCompat.getMainExecutor(this)) { imageProxy ->
                processImage(imageProxy)
            }

            try {
                cameraProvider.unbindAll()
                cameraProvider.bindToLifecycle(
                    this,
                    CameraSelector.DEFAULT_BACK_CAMERA,
                    preview,
                    imageAnalysis
                )
            } catch (e: Exception) {
                Log.e(MainConstants.TAG, "Camera start failed: ${e.message}")
                e.printStackTrace()
            }

        }, ContextCompat.getMainExecutor(this))
    }

    private fun processImage(image: ImageProxy) {
        val isConnected = mqttHandler?.isConnected() == true
        if (!isConnected) {
            image.close()
            return
        }

        val currentTime = System.currentTimeMillis()
        var proceedToSend = false

        if (isLiveMode) {
            if (currentTime - lastSentTime >= MainConstants.LIVE_MODE_INTERVAL_MS) {
                proceedToSend = true
                lastSentTime = currentTime
            }
        } else {
            if (isAutoSending) {
                if (currentTime - lastSentTime >= autoSendIntervalMillis) {
                    proceedToSend = true
                    lastSentTime = currentTime
                }
            } else if (shouldCaptureImage) {
                appendLog(getString(R.string.main_logs_image_capture_requested_entry))
                proceedToSend = true
                shouldCaptureImage = false
            }
        }

        if (!proceedToSend) {
            image.close()
            return
        }

        try {
            val rotationDegrees = image.imageInfo.rotationDegrees
            val nv21ImageRaw = ImageUtils.yuv420ToNv21(image)
            val nv21Image = if (rotationDegrees != 0) {
                ImageUtils.rotateNV21(nv21ImageRaw, rotationDegrees)
            } else {
                nv21ImageRaw
            }

            val yuvImage = YuvImage(nv21Image.bytes, ImageFormat.NV21, nv21Image.width, nv21Image.height, null)
            val out = ByteArrayOutputStream()

            yuvImage.compressToJpeg(
                Rect(0, 0, nv21Image.width, nv21Image.height),
                currentQuality,
                out
            )
            val jpegBytes = out.toByteArray()

            image.close()

            lifecycleScope.launch {
                mqttHandler?.publishImage(jpegBytes)
                if (!isLiveMode) {
                    appendLog(getString(R.string.main_logs_image_sent_entry, jpegBytes.size / 1024.0))
                }
            }

        } catch (e: Exception) {
            appendLog(getString(R.string.main_logs_image_capture_error_entry))
            Log.e(MainConstants.TAG, "Error processing image: ${e.message}")
            e.printStackTrace()
            try { image.close() } catch (_: Exception) {}
        }
    }

    private fun appendLog(message: String) {
        val timestamp = SimpleDateFormat("HH:mm:ss", Locale.getDefault()).format(Date())
        val currentText = logsTextView.text.toString()
        val lines = currentText.split("\n").takeLast(9)
        logsTextView.text = getString(
            R.string.main_logs_entry_format,
            lines.joinToString("\n"),
            timestamp,
            message
        )
    }

    override fun onDestroy() {
        super.onDestroy()

        if (isFinishing) {
            ProcessLifecycleOwner.get().lifecycleScope.launch {
                mqttHandler?.disconnect()
            }
        }
    }
}