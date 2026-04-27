package ro.pub.cs.systems.ssproject

import org.junit.Test

import org.junit.Assert.*
import ro.pub.cs.systems.ssproject.utils.ImageUtils
import ro.pub.cs.systems.ssproject.utils.ProcessedImage

/**
 * Example local unit test, which will execute on the development machine (host).
 *
 * See [testing documentation](http://d.android.com/tools/testing).
 */
class ExampleUnitTest {
    @Test
    fun rotateNV21_90Degrees_SwapsDimensions() {
        // Create a sample image and process it
        val width = 4
        val height = 2
        val data = ByteArray((width * height * 1.5).toInt())
        val image = ProcessedImage(data, width, height)

        // Rotate the image by 90 degrees
        val rotated = ImageUtils.rotateNV21(image, 90)

        assertEquals(height, rotated.width)
        assertEquals(width, rotated.height)
    }
}