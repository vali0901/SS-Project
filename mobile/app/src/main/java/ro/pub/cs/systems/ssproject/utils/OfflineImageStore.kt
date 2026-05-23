package ro.pub.cs.systems.ssproject.utils

import android.content.Context
import android.util.Log
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import java.io.File
import java.util.Locale

class OfflineImageStore(context: Context) {
    private val queueDir = File(context.filesDir, OFFLINE_QUEUE_DIR)

    suspend fun save(imageBytes: ByteArray): Int = withContext(Dispatchers.IO) {
        if (!queueDir.exists() && !queueDir.mkdirs()) {
            Log.e(TAG, "Could not create offline queue directory: ${queueDir.absolutePath}")
            return@withContext queuedCount()
        }

        trimQueueLocked()

        val imageFile = File(queueDir, buildFileName())
        val tmpFile = File(queueDir, "${imageFile.name}.tmp")

        try {
            tmpFile.writeBytes(imageBytes)
            if (!tmpFile.renameTo(imageFile)) {
                tmpFile.delete()
                Log.e(TAG, "Could not persist offline image: ${imageFile.absolutePath}")
            }
        } catch (e: Exception) {
            tmpFile.delete()
            Log.e(TAG, "Error saving offline image: ${e.message}")
        }

        queuedCount()
    }

    suspend fun flushNext(send: suspend (ByteArray) -> Boolean): FlushResult = withContext(Dispatchers.IO) {
        val imageFile = nextImageFile() ?: return@withContext FlushResult(false, queuedCount())

        val sent = try {
            send(imageFile.readBytes())
        } catch (e: Exception) {
            Log.e(TAG, "Error sending offline image: ${e.message}")
            false
        }

        if (sent && !imageFile.delete()) {
            Log.w(TAG, "Sent offline image but could not delete: ${imageFile.absolutePath}")
        }

        FlushResult(sent, queuedCount())
    }

    fun queuedCount(): Int {
        return queueDir
            .listFiles { file -> file.isFile && file.extension == IMAGE_EXTENSION }
            ?.size ?: 0
    }

    private fun nextImageFile(): File? {
        return queueDir
            .listFiles { file -> file.isFile && file.extension == IMAGE_EXTENSION }
            ?.minByOrNull { it.name }
    }

    private fun trimQueueLocked() {
        val files = queueDir
            .listFiles { file -> file.isFile && file.extension == IMAGE_EXTENSION }
            ?.sortedBy { it.name }
            ?.toMutableList()
            ?: return

        while (files.size >= MAX_OFFLINE_IMAGES) {
            files.removeAt(0).delete()
        }
    }

    private fun buildFileName(): String {
        return String.format(
            Locale.US,
            "image_%013d_%04d.%s",
            System.currentTimeMillis(),
            (0..9999).random(),
            IMAGE_EXTENSION
        )
    }

    data class FlushResult(
        val sent: Boolean,
        val remaining: Int
    )

    private companion object {
        const val TAG = "OfflineImageStore"
        const val OFFLINE_QUEUE_DIR = "offline_images"
        const val IMAGE_EXTENSION = "jpg"
        const val MAX_OFFLINE_IMAGES = 100
    }
}
