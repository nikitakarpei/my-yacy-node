plugins {
    id("companion.android-library")
}

android {
    namespace = "io.github.nikitakarpei.yacypeercompanion.services"
}

val goLibrary = layout.buildDirectory.file("go/pageservices.aar")
val goUnpacked = layout.buildDirectory.dir("go/unpacked")

val bindGo =
    tasks.register<Exec>("bindGo") {
        description = "Binds the Go packages of the app into an Android library with gomobile."
        group = "build"
        workingDir = rootProject.projectDir
        commandLine(
            rootProject.file("bind-go").path,
            goLibrary.get().asFile.absolutePath,
            APP_MIN_SDK.toString(),
            pinnedVersionOf("android-ndk"),
            "pairing",
            "pageservices",
        )
        outputs.file(goLibrary)
        outputs.upToDateWhen { false }
    }

val unpackGo =
    tasks.register<Sync>("unpackGo") {
        description = "Unpacks the Go library into its classes and native code, which stay inside this module."
        group = "build"
        from(bindGo.map { zipTree(goLibrary) })
        into(goUnpacked)
    }

android.sourceSets.getByName("main").jniLibs.directories.add(goUnpacked.get().dir("jni").asFile.path)

tasks.named("preBuild") { dependsOn(unpackGo) }

dependencies {
    api(project(":model"))
    implementation(project(":httpcapture"))
    implementation(files(goUnpacked.map { it.file("classes.jar") }).builtBy(unpackGo))
}
