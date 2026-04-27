package ro.pub.cs.systems.ssproject.utils

import android.content.Intent
import android.content.pm.PackageManager
import android.net.Uri
import android.provider.Settings
import androidx.activity.result.contract.ActivityResultContracts
import androidx.appcompat.app.AppCompatActivity
import androidx.core.app.ActivityCompat
import androidx.core.content.ContextCompat.checkSelfPermission
import com.google.android.material.dialog.MaterialAlertDialogBuilder
import ro.pub.cs.systems.ssproject.R

class PermissionHandler(
    private val activity: AppCompatActivity,
    private val permissions: Array<String>,
    private val rationaleTitle: Int,
    private val rationaleDescription: Int,
    private val settingsRedirectTitle: Int,
    private val settingsRedirectDescription: Int,
    private val onPermissionsGranted: () -> Unit,
    private val onPermissionsDenied: () -> Unit
) {
    private val requestPermissionsLauncher = activity.registerForActivityResult(
        ActivityResultContracts.RequestMultiplePermissions()
    ) { permissionsMap ->
        val allGranted = permissionsMap.values.all { it }

        if (allGranted) {
            onPermissionsGranted()
        } else {
            checkIfPermanentlyDenied()
        }
    }

    private val settingsLauncher = activity.registerForActivityResult(
        ActivityResultContracts.StartActivityForResult()
    ) {
        if (arePermissionsGranted()) {
            onPermissionsGranted()
        } else {
            onPermissionsDenied()
        }
    }

    fun requestPermissions() {
        when {
            arePermissionsGranted() -> {
                onPermissionsGranted()
            }

            shouldShowRationale() -> {
                showRationaleDialog()
            }

            else -> {
                requestPermissionsLauncher.launch(permissions)
            }
        }
    }

    private fun checkIfPermanentlyDenied() {
        val isPermanentlyDenied = permissions.any { permission ->
            checkSelfPermission(activity, permission) != PackageManager.PERMISSION_GRANTED &&
                    !ActivityCompat.shouldShowRequestPermissionRationale(activity, permission)
        }

        if (isPermanentlyDenied) {
            showSettingsDialog()
        } else {
            onPermissionsDenied()
        }
    }

    private fun arePermissionsGranted(): Boolean {
        return permissions.all { permission ->
            checkSelfPermission(activity, permission) == PackageManager.PERMISSION_GRANTED
        }
    }

    private fun shouldShowRationale(): Boolean {
        return permissions.any { permission ->
            ActivityCompat.shouldShowRequestPermissionRationale(activity, permission)
        }
    }

    private fun showRationaleDialog() {
        MaterialAlertDialogBuilder(activity)
            .setTitle(rationaleTitle)
            .setMessage(rationaleDescription)
            .setOnCancelListener { onPermissionsDenied() }
            .setNegativeButton(R.string.permission_rationale_negative_button) { dialog, _ ->
                dialog.dismiss()
                onPermissionsDenied()
            }
            .setPositiveButton(R.string.permission_rationale_positive_button) { _, _ ->
                requestPermissionsLauncher.launch(permissions)
            }
            .show()
    }

    private fun showSettingsDialog() {
        MaterialAlertDialogBuilder(activity)
            .setTitle(settingsRedirectTitle)
            .setMessage(settingsRedirectDescription)
            .setOnCancelListener { onPermissionsDenied() }
            .setNeutralButton(R.string.permissions_denied_settings_button) { _, _ ->
                val intent = Intent(
                    Settings.ACTION_APPLICATION_DETAILS_SETTINGS,
                    Uri.fromParts("package", activity.packageName, null)
                )
                settingsLauncher.launch(intent)
            }
            .setPositiveButton(R.string.permissions_denied_positive_button) { dialog, _ ->
                dialog.dismiss()
                onPermissionsDenied()
            }
            .show()
    }
}