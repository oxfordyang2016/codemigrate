{{ define "username" }} {{.TargetUser.Username}}, {{end}}
{{ define "content"}} Your package "{{.PkgEvent.Pkg.Title}}" has been uploaded completely.{{end}}
{{ define "detail_label"}}Details:{{end}}
{{ define "detail"}} 
    <p><span class="title">Total Size:</span> {{ .PkgEvent.Pkg.HumanSize }}B</p>
    <p><span class="title">Number of files:</span> {{ .PkgEvent.Pkg.NumFiles }}</p>
    <p><span class="title">Finished at:</span> {{ DatetimeFormat "en" .PkgEvent.Time }} UTC</p>
{{end}}
