package io.github.nikitakarpei.yacypeercompanion

import android.animation.ObjectAnimator
import android.animation.ValueAnimator
import android.content.Context
import android.graphics.Canvas
import android.graphics.Paint
import android.graphics.RectF
import android.view.View
import android.view.animation.LinearInterpolator

private const val SPIN_MS = 1_200L
private const val ARC_DEGREES = 90f

class ProgressRing(ctx: Context) : View(ctx) {
    private val stroke = ctx.dp(4).toFloat()
    private val bounds = RectF()
    private val track = paintOf(ctx.colorOf(R.color.primary_container))
    private val arc = paintOf(ctx.colorOf(R.color.primary))
    private val spin =
        ObjectAnimator.ofFloat(this, ROTATION, 0f, 360f).apply {
            duration = SPIN_MS
            repeatCount = ValueAnimator.INFINITE
            interpolator = LinearInterpolator()
        }

    private fun paintOf(color: Int) = Paint(Paint.ANTI_ALIAS_FLAG).apply {
        style = Paint.Style.STROKE
        strokeWidth = stroke
        strokeCap = Paint.Cap.ROUND
        this.color = color
    }

    override fun onSizeChanged(width: Int, height: Int, oldWidth: Int, oldHeight: Int) {
        super.onSizeChanged(width, height, oldWidth, oldHeight)
        bounds.set(stroke / 2, stroke / 2, width - stroke / 2, height - stroke / 2)
    }

    override fun onDraw(canvas: Canvas) {
        super.onDraw(canvas)
        canvas.drawOval(bounds, track)
        canvas.drawArc(bounds, -90f, ARC_DEGREES, false, arc)
    }

    override fun onAttachedToWindow() {
        super.onAttachedToWindow()
        spin.start()
    }

    override fun onDetachedFromWindow() {
        spin.cancel()
        super.onDetachedFromWindow()
    }
}
