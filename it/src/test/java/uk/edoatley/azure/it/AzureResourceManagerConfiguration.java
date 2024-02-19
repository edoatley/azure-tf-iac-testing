package uk.edoatley.azure.it;

import com.azure.core.management.AzureEnvironment;
import com.azure.core.management.profile.AzureProfile;
import com.azure.identity.DefaultAzureCredential;
import com.azure.identity.DefaultAzureCredentialBuilder;
import com.azure.resourcemanager.AzureResourceManager;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Configuration
public class AzureResourceManagerConfiguration {

    @Value("${azure.tenant}")
    private String tenant;

    @Value("${azure.subscription}")
    private String subscription;

    @Bean
    public AzureResourceManager azureResourceManager() {
        AzureProfile toolingAzureProfile = new AzureProfile(tenant, subscription, AzureEnvironment.AZURE);
        DefaultAzureCredential toolingCredential = new DefaultAzureCredentialBuilder()
                .authorityHost(toolingAzureProfile.getEnvironment().getActiveDirectoryEndpoint())
                .build();
        return AzureResourceManager
                .authenticate(toolingCredential, toolingAzureProfile)
                .withDefaultSubscription();
    }
}
