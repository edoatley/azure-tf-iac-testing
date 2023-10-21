package uk.edoatley.azure.it;

import java.util.Map;

import org.springframework.boot.context.properties.ConfigurationProperties;
import org.springframework.stereotype.Component;

import lombok.Data;

@Data
@Component
@ConfigurationProperties(prefix = "azure.config")
public class AzureConfigurationProperties {
    private String resourceGroupName;
    private String vnetName;
    private String vnetAddressSpace;
    private Map<String, String> subnets;
    private String importantIpAddress;
}
