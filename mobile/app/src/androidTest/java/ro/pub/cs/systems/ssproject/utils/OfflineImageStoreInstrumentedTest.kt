package ro.pub.cs.systems.ssproject.utils

import android.content.Context
import androidx.test.ext.junit.runners.AndroidJUnit4
import androidx.test.platform.app.InstrumentationRegistry
import kotlinx.coroutines.runBlocking
import org.junit.After
import org.junit.Assert.assertArrayEquals
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import org.junit.runner.RunWith
import java.io.File

@RunWith(AndroidJUnit4::class)
class OfflineImageStoreInstrumentedTest {
    private lateinit var context: Context
    private lateinit var queueDir: File
    private lateinit var store: OfflineImageStore

    @Before
    fun setUp() {
        context = InstrumentationRegistry.getInstrumentation().targetContext
        queueDir = File(context.filesDir, OFFLINE_QUEUE_DIR)
        queueDir.deleteRecursively()
        store = OfflineImageStore(context)
    }

    @After
    fun tearDown() {
        queueDir.deleteRecursively()
    }

    @Test
    fun save_persistsImageAndReportsQueuedCount() = runBlocking {
        val count = store.save(byteArrayOf(1, 2, 3))

        assertEquals(1, count)
        assertEquals(1, store.queuedCount())

        val savedFile = queueDir.listFiles()?.single()
        assertEquals("jpg", savedFile?.extension)
        assertArrayEquals(byteArrayOf(1, 2, 3), savedFile?.readBytes())
    }

    @Test
    fun flushNext_whenSendSucceeds_sendsOldestImageAndDeletesIt() = runBlocking {
        assertEquals(1, store.save(byteArrayOf(1)))
        Thread.sleep(2)
        assertEquals(2, store.save(byteArrayOf(2)))

        val sentPayloads = mutableListOf<ByteArray>()
        val result = store.flushNext { bytes ->
            sentPayloads += bytes
            true
        }

        assertTrue(result.sent)
        assertEquals(1, result.remaining)
        assertEquals(1, sentPayloads.size)
        assertArrayEquals(byteArrayOf(1), sentPayloads.single())
        assertEquals(1, store.queuedCount())
    }

    @Test
    fun flushNext_whenSendFails_keepsImageQueued() = runBlocking {
        assertEquals(1, store.save(byteArrayOf(4, 5, 6)))

        val result = store.flushNext { false }

        assertFalse(result.sent)
        assertEquals(1, result.remaining)
        assertEquals(1, store.queuedCount())
    }

    @Test
    fun save_trimsQueueToMaximumSize() = runBlocking {
        repeat(MAX_OFFLINE_IMAGES + 1) { index ->
            store.save(byteArrayOf(index.toByte()))
            Thread.sleep(1)
        }

        assertEquals(MAX_OFFLINE_IMAGES, store.queuedCount())
    }

    private companion object {
        const val OFFLINE_QUEUE_DIR = "offline_images"
        const val MAX_OFFLINE_IMAGES = 100
    }
}
