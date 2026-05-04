package ro.pub.cs.systems.ssproject.mqtt
 
import android.content.Context
import java.security.KeyStore
import javax.net.ssl.KeyManagerFactory
import javax.net.ssl.SSLContext
import javax.net.ssl.SSLSocketFactory
import javax.net.ssl.TrustManagerFactory
 
object TlsHelper {
 
    /**
     * Creează un SSLSocketFactory configurat pentru mTLS.
     *
     * @param context Contextul Android (necesar pentru accesarea resurselor raw)
     * @param trustStoreResId ID-ul resursei raw pentru truststore (R.raw.truststore)
     * @param keyStoreResId ID-ul resursei raw pentru keystore (R.raw.keystore)
     * @return SSLSocketFactory configurat cu certificatele client și CA
     */
    fun createMtlsSocketFactory(
        context: Context,
        trustStoreResId: Int,
        keyStoreResId: Int
    ): SSLSocketFactory {
        // 1. Încărcarea TrustStore-ului (conține certificatul CA)
        val trustStore = KeyStore.getInstance(MqttConstants.BKS_STORE_TYPE)
        context.resources.openRawResource(trustStoreResId).use { input ->
            trustStore.load(input, MqttConstants.TRUSTSTORE_PASSWORD.toCharArray())
        }
 
        val trustManagerFactory = TrustManagerFactory.getInstance(
            TrustManagerFactory.getDefaultAlgorithm()
        )
        trustManagerFactory.init(trustStore)
 
        // 2. Încărcarea KeyStore-ului (conține certificatul și cheia clientului)
        val keyStore = KeyStore.getInstance(MqttConstants.BKS_STORE_TYPE)
        context.resources.openRawResource(keyStoreResId).use { input ->
            keyStore.load(input, MqttConstants.KEYSTORE_PASSWORD.toCharArray())
        }
 
        val keyManagerFactory = KeyManagerFactory.getInstance(
            KeyManagerFactory.getDefaultAlgorithm()
        )
        keyManagerFactory.init(keyStore, MqttConstants.KEYSTORE_PASSWORD.toCharArray())
 
        // 3. Crearea contextului SSL cu ambele componente (mTLS)
        val sslContext = SSLContext.getInstance(MqttConstants.TLS_PROTOCOL)
        sslContext.init(
            keyManagerFactory.keyManagers,   // Autentificarea clientului
            trustManagerFactory.trustManagers, // Verificarea serverului
            null
        )
 
        return sslContext.socketFactory
    }
}