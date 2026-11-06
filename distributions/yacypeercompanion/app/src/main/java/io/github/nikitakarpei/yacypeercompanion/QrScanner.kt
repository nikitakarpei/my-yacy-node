package io.github.nikitakarpei.yacypeercompanion

import android.annotation.SuppressLint
import android.content.Context
import android.graphics.ImageFormat
import android.graphics.Matrix
import android.graphics.SurfaceTexture
import android.hardware.camera2.CameraCaptureSession
import android.hardware.camera2.CameraCharacteristics
import android.hardware.camera2.CameraDevice
import android.hardware.camera2.CameraManager
import android.hardware.camera2.CaptureRequest
import android.hardware.camera2.params.OutputConfiguration
import android.hardware.camera2.params.SessionConfiguration
import android.media.Image
import android.media.ImageReader
import android.os.Handler
import android.os.HandlerThread
import android.util.Size
import android.view.Surface
import android.view.TextureView
import com.google.zxing.BinaryBitmap
import com.google.zxing.DecodeHintType
import com.google.zxing.PlanarYUVLuminanceSource
import com.google.zxing.ReaderException
import com.google.zxing.common.HybridBinarizer
import com.google.zxing.qrcode.QRCodeReader
import java.util.concurrent.Executor

private const val LARGEST_FRAME_WIDTH = 1280

class QrScanner(private val ctx: Context, private val preview: TextureView, private val onText: (String) -> Unit) {
    private val cameraThread = HandlerThread("qr-scanner").apply { start() }
    private val cameraHandler = Handler(cameraThread.looper)
    private val cameraExecutor = Executor { cameraHandler.post(it) }
    private val reader = QRCodeReader()
    private var camera: CameraDevice? = null
    private var frames: ImageReader? = null

    @Volatile
    private var paused = false

    @SuppressLint("MissingPermission")
    fun start() {
        val manager = ctx.getSystemService(CameraManager::class.java)
        val cameraId = backCameraOf(manager)
        val size = frameSizeOf(manager.getCameraCharacteristics(cameraId))
        cropToFill(size)
        val frames = ImageReader.newInstance(size.width, size.height, ImageFormat.YUV_420_888, 2)
        frames.setOnImageAvailableListener({ reader -> reader.acquireLatestImage()?.use { read(it) } }, cameraHandler)
        this.frames = frames
        manager.openCamera(
            cameraId,
            object : CameraDevice.StateCallback() {
                override fun onOpened(device: CameraDevice) {
                    camera = device
                    startPreview(device, size, frames)
                }

                override fun onDisconnected(device: CameraDevice) = device.close()

                override fun onError(device: CameraDevice, error: Int) = device.close()
            },
            cameraHandler,
        )
    }

    private fun cropToFill(size: Size) {
        val viewWidth = preview.width.toFloat()
        val viewHeight = preview.height.toFloat()
        val pictureAspect = size.height.toFloat() / size.width
        val viewAspect = viewWidth / viewHeight
        val crop = Matrix()
        if (viewAspect > pictureAspect) {
            crop.setScale(1f, viewAspect / pictureAspect, viewWidth / 2, viewHeight / 2)
        } else {
            crop.setScale(pictureAspect / viewAspect, 1f, viewWidth / 2, viewHeight / 2)
        }
        preview.setTransform(crop)
    }

    private fun backCameraOf(manager: CameraManager): String = manager.cameraIdList.firstOrNull {
        manager.getCameraCharacteristics(it).get(CameraCharacteristics.LENS_FACING) == CameraCharacteristics.LENS_FACING_BACK
    } ?: manager.cameraIdList.first()

    private fun frameSizeOf(characteristics: CameraCharacteristics): Size {
        val map = checkNotNull(characteristics.get(CameraCharacteristics.SCALER_STREAM_CONFIGURATION_MAP))
        val previewSizes = map.getOutputSizes(SurfaceTexture::class.java).toSet()
        val sizes = map.getOutputSizes(ImageFormat.YUV_420_888).filter { it in previewSizes && it.width > it.height }
        return sizes.filter { it.width <= LARGEST_FRAME_WIDTH }.maxByOrNull { it.width * it.height } ?: sizes.first()
    }

    private fun startPreview(device: CameraDevice, size: Size, frames: ImageReader) {
        val texture = checkNotNull(preview.surfaceTexture).apply { setDefaultBufferSize(size.width, size.height) }
        val previewSurface = Surface(texture)
        val request =
            device.createCaptureRequest(CameraDevice.TEMPLATE_PREVIEW).apply {
                addTarget(previewSurface)
                addTarget(frames.surface)
                set(CaptureRequest.CONTROL_AF_MODE, CaptureRequest.CONTROL_AF_MODE_CONTINUOUS_PICTURE)
            }
        device.createCaptureSession(
            SessionConfiguration(
                SessionConfiguration.SESSION_REGULAR,
                listOf(OutputConfiguration(previewSurface), OutputConfiguration(frames.surface)),
                cameraExecutor,
                object : CameraCaptureSession.StateCallback() {
                    override fun onConfigured(session: CameraCaptureSession) {
                        session.setRepeatingRequest(request.build(), null, cameraHandler)
                    }

                    override fun onConfigureFailed(session: CameraCaptureSession) = session.close()
                },
            ),
        )
    }

    private fun read(image: Image) {
        if (paused) return
        val luminance = image.planes[0]
        val bytes = ByteArray(luminance.buffer.remaining()).also { luminance.buffer.get(it) }
        val rows = bytes.size / luminance.rowStride + 1
        val source = PlanarYUVLuminanceSource(bytes, luminance.rowStride, rows, 0, 0, image.width, image.height, false)
        val text =
            try {
                reader.decode(BinaryBitmap(HybridBinarizer(source)), mapOf(DecodeHintType.TRY_HARDER to true)).text
            } catch (e: ReaderException) {
                return
            } finally {
                reader.reset()
            }
        paused = true
        onText(text)
    }

    fun pause() {
        paused = true
    }

    fun resume() {
        paused = false
    }

    fun stop() {
        camera?.close()
        frames?.close()
        cameraThread.quitSafely()
    }
}
