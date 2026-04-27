package ro.pub.cs.systems.ssproject.utils

import androidx.camera.core.ImageProxy

data class ProcessedImage(
    val bytes: ByteArray,
    val width: Int,
    val height: Int
) {
    override fun equals(other: Any?): Boolean {
        if (this === other) return true
        if (javaClass != other?.javaClass) return false

        other as ProcessedImage

        if (width != other.width) return false
        if (height != other.height) return false
        if (!bytes.contentEquals(other.bytes)) return false

        return true
    }

    override fun hashCode(): Int {
        var result = width
        result = 31 * result + height
        result = 31 * result + bytes.contentHashCode()
        return result
    }
}

object ImageUtils {
    fun yuv420ToNv21(image: ImageProxy): ProcessedImage {
        val width = image.width
        val height = image.height

        val ySize = width * height
        val uvSize = width * height / 2
        val pixelCount = ySize + uvSize

        val nv21 = ByteArray(pixelCount)

        val yPlane = image.planes[0]
        val uPlane = image.planes[1]
        val vPlane = image.planes[2]

        val yBuffer = yPlane.buffer
        val uBuffer = uPlane.buffer
        val vBuffer = vPlane.buffer

        yBuffer.rewind()
        uBuffer.rewind()
        vBuffer.rewind()

        val yRowStride = yPlane.rowStride

        if (yRowStride == width) {
            yBuffer.get(nv21, 0, ySize)
        } else {
            var pos = 0
            for (row in 0 until height) {
                yBuffer.position(row * yRowStride)
                yBuffer.get(nv21, pos, width)
                pos += width
            }
        }

        val rowStride = uPlane.rowStride
        val pixelStride = uPlane.pixelStride

        var pos = ySize

        val uBufferLimit = uBuffer.limit()
        val vBufferLimit = vBuffer.limit()

        for (row in 0 until height / 2) {
            for (col in 0 until width / 2) {
                val index = row * rowStride + col * pixelStride

                if (index < vBufferLimit) {
                    nv21[pos++] = vBuffer.get(index)
                }

                if (index < uBufferLimit) {
                    nv21[pos++] = uBuffer.get(index)
                }
            }
        }

        return ProcessedImage(nv21, width, height)
    }

    fun rotateNV21(image: ProcessedImage, rotation: Int): ProcessedImage {
        if (rotation == 0) return image

        val input = image.bytes
        val width = image.width
        val height = image.height
        val output = ByteArray(input.size)

        val frameSize = width * height
        val swapDimensions = (rotation == 90 || rotation == 270)

        val newWidth = if (swapDimensions) height else width
        val newHeight = if (swapDimensions) width else height

        var k = 0

        when (rotation) {
            90 -> {
                for (col in 0 until width) {
                    for (row in height - 1 downTo 0) {
                        output[k++] = input[row * width + col]  // Y
                    }
                }
                for (col in 0 until width step 2) {
                    for (row in height / 2 - 1 downTo 0) {
                        val offset = frameSize + (row * width) + col
                        output[k++] = input[offset]             // V
                        output[k++] = input[offset + 1]         // U
                    }
                }
            }
            180 -> {
                for (i in frameSize - 1 downTo 0) {
                    output[k++] = input[i]      // Y
                }
                for (i in input.size - 1 downTo frameSize step 2) {
                    output[k++] = input[i]      // V
                    output[k++] = input[i - 1]  // U
                }
            }
            270 -> {
                for (col in width - 1 downTo 0) {
                    for (row in 0 until height) {
                        output[k++] = input[row * width + col]  //Y
                    }
                }
                for (col in width - 1 downTo 0 step 2) {
                    for (row in 0 until height / 2) {
                        val offset = frameSize + (row * width) + (col - 1)
                        output[k++] = input[offset]             // V
                        output[k++] = input[offset + 1]         // U
                    }
                }
            }
        }

        return ProcessedImage(output, newWidth, newHeight)
    }
}