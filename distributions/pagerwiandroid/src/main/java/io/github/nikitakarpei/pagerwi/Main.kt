package io.github.nikitakarpei.pagerwi

import io.github.nikitakarpei.servicehost.ServiceApp
import io.github.nikitakarpei.servicehost.ServiceHost
import io.github.nikitakarpei.servicehost.ServiceInstance
import io.github.nikitakarpei.servicehost.ServiceScreen
import pagerwi.Pagerwi

object PagerwiApp : ServiceApp("pagerwi") {
    override val hostClass = PagerwiHost::class.java

    override val screenClass = MainActivity::class.java

    override fun variables(): String = Pagerwi.variables()

    override fun instanceFrom(environment: String): ServiceInstance {
        val instance = Pagerwi.instanceFrom(environment)
        return object : ServiceInstance {
            override fun run() = instance.run()

            override fun stop() = instance.stop()
        }
    }
}

class PagerwiHost : ServiceHost() {
    override val app = PagerwiApp
}

class MainActivity : ServiceScreen() {
    override val app = PagerwiApp
}
