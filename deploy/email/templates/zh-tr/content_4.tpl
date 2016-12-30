{{ define "username" }} 您好，{{.TargetUser.Username}}: {{end}}
{{ define "content"}} 您發送的包裹"{{.PkgEvent.Pkg.Title}}", 接收者"{{.PkgEvent.User.Username}}"已完成下載{{end}}
{{ define "detail_label"}}詳情:{{end}}
{{ define "detail"}} 
    <p><span class="title">包裹大小:</span> {{ .PkgEvent.Pkg.HumanSize }}B</p>
    <p><span class="title">文件個數:</span> {{ .PkgEvent.Pkg.NumFiles }}</p>
    <p><span class="title">完成時間:</span> 北京時間 {{ DatetimeFormat "zh" .PkgEvent.Time }}</p>
{{end}}
