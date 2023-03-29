package openapi

var swaggerUiDefaultParameters = map[string]string{
	"dom_id":               `"#swagger-ui"`,
	"deepLinking":          "true",
	"showExtensions":       "true",
	"showCommonExtensions": "true",
}

var swaggerUiHtml = ""
var redocUiHtml = ""
var oauthUiHtml = ""

func MakeSwaggerUiHtml(title, openapiUrl, jsUrl, cssUrl, faviconUrl string) string {
	if len(swaggerUiHtml) < 1 {
		indexPage := `
	<!DOCTYPE html>
	<html>
	<head>
		<link type="text/css" rel="stylesheet" href="` + cssUrl + `">
		<link rel="shortcut icon" href="` + faviconUrl + `">
		<title>` + title + ` - Swagger UI</title>
	</head>
	<body>
		<div id="swagger-ui">
		</div>
		<script src="` + jsUrl + `"></script>
		<script>
		const ui = SwaggerUIBundle({
		url: './` + openapiUrl + `',
	`
		for k, v := range swaggerUiDefaultParameters {
			indexPage = indexPage + `"` + k + `": ` + v + ",\n"
		}

		indexPage = indexPage + "oauth2RedirectUrl: window.location.origin + '/docs/oauth2-redirect',\n"

		indexPage += ` presets: [
        SwaggerUIBundle.presets.apis,
        SwaggerUIBundle.SwaggerUIStandalonePreset
        ],
    })
	</script>
    </body>
    </html>
	`
		swaggerUiHtml = indexPage
	}

	return swaggerUiHtml
}

func MakeRedocUiHtml(title, openapiUrl, jsUrl, faviconUrl string) string {
	if len(redocUiHtml) < 1 {
		indexPage := `
	<!DOCTYPE html>
	<html>
	<head>= 
		<title>` + title + ` </title>
		<!-- needed for adaptive design -->
		<meta charset="utf-8"/>
		<meta name="viewport" content="width=device-width, initial-scale=1">
		<link href="https://fonts.googleapis.com/css?family=Montserrat:300,400,700|Roboto:300,400,700" rel="stylesheet">
		<link rel="shortcut icon" href="` + faviconUrl + `">
		<!--
			ReDoc doesn't change outer page styles
		-->
		<style>
			body {{
				margin: 0;
				padding: 0;
			}}
		</style>
	</head>`

		indexPage += `
	<body>
	<noscript>
		ReDoc requires Javascript to function. Please enable it to browse the documentation.
	</noscript>
	<redoc spec-url="` + openapiUrl + `"></redoc>
	<script src="` + jsUrl + `"> </script>
	</body>
	</html>`

		redocUiHtml = indexPage
	}

	return redocUiHtml
}

func MakeOauth2RedirectHtml() string {
	// copied from https://github.com/swagger-api/swagger-ui/blob/v4.14.0/dist/oauth2-redirect.html
	if len(oauthUiHtml) < 1 {
		oauthUiHtml = `    
    html = """
    <!doctype html>
    <html lang="en-US">
    <head>
        <title>Swagger UI: OAuth2 Redirect</title>
    </head>
    <body>
    <script>
        'use strict';
        function run () {
            var oauth2 = window.opener.swaggerUIRedirectOauth2;
            var sentState = oauth2.state;
            var redirectUrl = oauth2.redirectUrl;
            var isValid, qp, arr;

            if (/code|token|error/.example(window.location.hash)) {
                qp = window.location.hash.substring(1).replace('?', '&');
            } else {
                qp = location.search.substring(1);
            }

            arr = qp.split("&");
            arr.forEach(function (v,i,_arr) { _arr[i] = '"' + v.replace('=', '":"') + '"';});
            qp = qp ? JSON.parse('{' + arr.join() + '}',
                    function (key, value) {
                        return key === "" ? value : decodeURIComponent(value);
                    }
            ) : {};

            isValid = qp.state === sentState;

            if ((
              oauth2.auth.schema.get("flow") === "accessCode" ||
              oauth2.auth.schema.get("flow") === "authorizationCode" ||
              oauth2.auth.schema.get("flow") === "authorization_code"
            ) && !oauth2.auth.code) {
                if (!isValid) {
                    oauth2.errCb({
                        authId: oauth2.auth.name,
                        source: "auth",
                        level: "warning",
                        message: "Authorization may be unsafe, passed state was changed in server. The passed state wasn't returned from auth server."
                    });
                }

                if (qp.code) {
                    delete oauth2.state;
                    oauth2.auth.code = qp.code;
                    oauth2.callback({auth: oauth2.auth, redirectUrl: redirectUrl});
                } else {
                    let oauthErrorMsg;
                    if (qp.error) {
                        oauthErrorMsg = "["+qp.error+"]: " +
                            (qp.error_description ? qp.error_description+ ". " : "no accessCode received from the server. ") +
                            (qp.error_uri ? "More info: "+qp.error_uri : "");
                    }

                    oauth2.errCb({
                        authId: oauth2.auth.name,
                        source: "auth",
                        level: "error",
                        message: oauthErrorMsg || "[Authorization failed]: no accessCode received from the server."
                    });
                }
            } else {
                oauth2.callback({auth: oauth2.auth, token: qp, isValid: isValid, redirectUrl: redirectUrl});
            }
            window.close();
        }

        if (document.readyState !== 'loading') {
            run();
        } else {
            document.addEventListener('DOMContentLoaded', function () {
                run();
            });
        }
    </script>
    </body>
    </html>`
	}

	return oauthUiHtml
}
