package ro.pub.cs.systems.ssproject.utils

import org.junit.Assert.assertArrayEquals
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotEquals
import org.junit.Assert.assertSame
import org.junit.Test

class ImageUtilsTest {
    @Test
    fun rotateNV21_zeroDegrees_returnsSameImage() {
        val image = ProcessedImage(
            bytes = byteArrayOf(1, 2, 3, 4, 5, 6),
            width = 2,
            height = 2
        )

        val rotated = ImageUtils.rotateNV21(image, 0)

        assertSame(image, rotated)
    }

    @Test
    fun rotateNV21_90Degrees_rotatesLumaAndChromaAndSwapsDimensions() {
        val image = sampleNv21Image()

        val rotated = ImageUtils.rotateNV21(image, 90)

        assertEquals(2, rotated.width)
        assertEquals(4, rotated.height)
        assertArrayEquals(
            byteArrayOf(
                5, 1,
                6, 2,
                7, 3,
                8, 4,
                9, 10,
                11, 12
            ),
            rotated.bytes
        )
    }

    @Test
    fun rotateNV21_180Degrees_rotatesLumaAndChroma() {
        val image = sampleNv21Image()

        val rotated = ImageUtils.rotateNV21(image, 180)

        assertEquals(4, rotated.width)
        assertEquals(2, rotated.height)
        assertArrayEquals(
            byteArrayOf(
                8, 7, 6, 5,
                4, 3, 2, 1,
                12, 11, 10, 9
            ),
            rotated.bytes
        )
    }

    @Test
    fun rotateNV21_270Degrees_rotatesLumaAndChromaAndSwapsDimensions() {
        val image = sampleNv21Image()

        val rotated = ImageUtils.rotateNV21(image, 270)

        assertEquals(2, rotated.width)
        assertEquals(4, rotated.height)
        assertArrayEquals(
            byteArrayOf(
                4, 8,
                3, 7,
                2, 6,
                1, 5,
                11, 12,
                9, 10
            ),
            rotated.bytes
        )
    }

    @Test
    fun processedImage_equalityUsesByteContentAndDimensions() {
        val image = ProcessedImage(byteArrayOf(1, 2, 3), width = 1, height = 2)
        val sameContent = ProcessedImage(byteArrayOf(1, 2, 3), width = 1, height = 2)
        val differentBytes = ProcessedImage(byteArrayOf(1, 2, 4), width = 1, height = 2)
        val differentDimensions = ProcessedImage(byteArrayOf(1, 2, 3), width = 3, height = 1)

        assertEquals(image, sameContent)
        assertEquals(image.hashCode(), sameContent.hashCode())
        assertNotEquals(image, differentBytes)
        assertNotEquals(image, differentDimensions)
    }

    private fun sampleNv21Image(): ProcessedImage {
        return ProcessedImage(
            bytes = byteArrayOf(
                1, 2, 3, 4,
                5, 6, 7, 8,
                9, 10, 11, 12
            ),
            width = 4,
            height = 2
        )
    }
}
