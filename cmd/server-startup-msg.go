/*
 * Minio Cloud Storage, (C) 2016, 2017 Minio, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cmd

import (
	"crypto/x509"
	"fmt"
	"runtime"
	"strings"

	humanize "github.com/dustin/go-humanize"
)

// Documentation links, these are part of message printing code.
const (
	mcQuickStartGuide   = "https://docs.minio.io/docs/minio-client-quickstart-guide"
	goQuickStartGuide   = "https://docs.minio.io/docs/golang-client-quickstart-guide"
	jsQuickStartGuide   = "https://docs.minio.io/docs/javascript-client-quickstart-guide"
	javaQuickStartGuide = "https://docs.minio.io/docs/java-client-quickstart-guide"
	pyQuickStartGuide   = "https://docs.minio.io/docs/python-client-quickstart-guide"
)

// generates format string depending on the string length and padding.
func getFormatStr(strLen int, padding int) string {
	formatStr := fmt.Sprintf("%ds", strLen+padding)
	return "%" + formatStr
}

// Prints the formatted startup message.
func printStartupMessage(apiEndPoints []string) {
	// Prints credential, region and browser access.
	printServerCommonMsg(apiEndPoints)

	// Prints `mc` cli configuration message chooses
	// first endpoint as default.
	printCLIAccessMsg(apiEndPoints[0])

	// Prints documentation message.
	printObjectAPIMsg()

	// Object layer is initialized then print StorageInfo.
	objAPI := newObjectLayerFn()
	if objAPI != nil {
		printStorageInfo(objAPI.StorageInfo())
	}

	// SSL is configured reads certification chain, prints
	// authority and expiry.
	if globalIsSSL {
		printCertificateMsg(globalPublicCerts)
	}
}

// Prints common server startup message. Prints credential, region and browser access.
func printServerCommonMsg(apiEndpoints []string) {
	// Get saved credentials.
	cred := serverConfig.GetCredential()

	// Get saved region.
	region := serverConfig.GetRegion()

	apiEndpointStr := strings.Join(apiEndpoints, "  ")
	// Colorize the message and print.
	log.Println(colorBlue("\nEndpoint: ") + colorBold(fmt.Sprintf(getFormatStr(len(apiEndpointStr), 1), apiEndpointStr)))
	log.Println(colorBlue("AccessKey: ") + colorBold(fmt.Sprintf("%s ", cred.AccessKey)))
	log.Println(colorBlue("SecretKey: ") + colorBold(fmt.Sprintf("%s ", cred.SecretKey)))
	log.Println(colorBlue("Region: ") + colorBold(fmt.Sprintf(getFormatStr(len(region), 3), region)))
	printEventNotifiers()

	log.Println(colorBlue("\nBrowser Access:"))
	log.Println(fmt.Sprintf(getFormatStr(len(apiEndpointStr), 3), apiEndpointStr))
}

// Prints bucket notification configurations.
func printEventNotifiers() {
	if globalEventNotifier == nil {
		// In case initEventNotifier() was not done or failed.
		return
	}
	arnMsg := colorBlue("SQS ARNs: ")
	// Get all configured external notification targets
	externalTargets := globalEventNotifier.GetAllExternalTargets()
	if len(externalTargets) == 0 {
		arnMsg += colorBold(fmt.Sprintf(getFormatStr(len("<none>"), 1), "<none>"))
	}
	for queueArn := range externalTargets {
		arnMsg += colorBold(fmt.Sprintf(getFormatStr(len(queueArn), 1), queueArn))
	}
	log.Println(arnMsg)
}

// Prints startup message for command line access. Prints link to our documentation
// and custom platform specific message.
func printCLIAccessMsg(endPoint string) {
	// Get saved credentials.
	cred := serverConfig.GetCredential()

	// Configure 'mc', following block prints platform specific information for minio client.
	log.Println(colorBlue("\nCommand-line Access: ") + mcQuickStartGuide)
	if runtime.GOOS == globalWindowsOSName {
		mcMessage := fmt.Sprintf("$ mc.exe config host add myminio %s %s %s", endPoint, cred.AccessKey, cred.SecretKey)
		log.Println(fmt.Sprintf(getFormatStr(len(mcMessage), 3), mcMessage))
	} else {
		mcMessage := fmt.Sprintf("$ mc config host add myminio %s %s %s", endPoint, cred.AccessKey, cred.SecretKey)
		log.Println(fmt.Sprintf(getFormatStr(len(mcMessage), 3), mcMessage))
	}
}

// Prints startup message for Object API acces, prints link to our SDK documentation.
func printObjectAPIMsg() {
	log.Println(colorBlue("\nObject API (Amazon S3 compatible):"))
	log.Println(colorBlue("   Go: ") + fmt.Sprintf(getFormatStr(len(goQuickStartGuide), 8), goQuickStartGuide))
	log.Println(colorBlue("   Java: ") + fmt.Sprintf(getFormatStr(len(javaQuickStartGuide), 6), javaQuickStartGuide))
	log.Println(colorBlue("   Python: ") + fmt.Sprintf(getFormatStr(len(pyQuickStartGuide), 4), pyQuickStartGuide))
	log.Println(colorBlue("   JavaScript: ") + jsQuickStartGuide)
}

// Get formatted disk/storage info message.
func getStorageInfoMsg(storageInfo StorageInfo) string {
	msg := fmt.Sprintf("%s %s Free, %s Total", colorBlue("Drive Capacity:"),
		humanize.IBytes(uint64(storageInfo.Free)),
		humanize.IBytes(uint64(storageInfo.Total)))
	if storageInfo.Backend.Type == Erasure {
		diskInfo := fmt.Sprintf(" %d Online, %d Offline. ", storageInfo.Backend.OnlineDisks, storageInfo.Backend.OfflineDisks)
		if maxDiskFailures := storageInfo.Backend.ReadQuorum - storageInfo.Backend.OfflineDisks; maxDiskFailures >= 0 {
			diskInfo += fmt.Sprintf("We can withstand [%d] more drive failure(s).", maxDiskFailures)
		}
		msg += colorBlue("\nStatus:") + fmt.Sprintf(getFormatStr(len(diskInfo), 8), diskInfo)
	}
	return msg
}

// Prints startup message of storage capacity and erasure information.
func printStorageInfo(storageInfo StorageInfo) {
	log.Println()
	log.Println(getStorageInfoMsg(storageInfo))
}

// Prints certificate expiry date warning
func getCertificateChainMsg(certs []*x509.Certificate) string {
	msg := colorBlue("\nCertificate expiry info:\n")
	totalCerts := len(certs)
	var expiringCerts int
	for i := totalCerts - 1; i >= 0; i-- {
		cert := certs[i]
		if cert.NotAfter.Before(UTCNow().Add(globalMinioCertExpireWarnDays)) {
			expiringCerts++
			msg += fmt.Sprintf(colorBold("#%d %s will expire on %s\n"), expiringCerts, cert.Subject.CommonName, cert.NotAfter)
		}
	}
	if expiringCerts > 0 {
		return msg
	}
	return ""
}

// Prints the certificate expiry message.
func printCertificateMsg(certs []*x509.Certificate) {
	log.Println(getCertificateChainMsg(certs))
}
