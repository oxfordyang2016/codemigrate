{{ define "username" }} {{.TargetUser.Username}}, {{end}}
{{ define "content"}} You have a new package "{{.PkgEvent.Pkg.Title}}" from {{.PkgEvent.Owner.Username}}.{{end}}
{{ define "remark_label"}}Remarks:{{end}}
{{ define "remark"}}{{.PkgEvent.Pkg.Notes}} {{end}}
{{ define "detail_label"}}Details:{{end}}
{{ define "detail"}} 
    <p><span class="title">Total Size:</span> {{ .PkgEvent.Pkg.HumanSize }}B</p>
    <p><span class="title">Number of files:</span> {{ .PkgEvent.Pkg.NumFiles }}</p>
    <p><span class="title">Created at:</span> {{ DatetimeFormat "en" .PkgEvent.Pkg.CreateAt }} UTC</p>
    <p><span class="title">Sender:</span> {{ .PkgEvent.Owner.Username }} </p>
{{end}}
