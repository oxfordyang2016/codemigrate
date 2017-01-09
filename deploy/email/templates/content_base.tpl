<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <meta http-equiv="X-UA-Compatible" content="ie=edge">
  <title>Cydex Notification</title>
  <style>
    html,body {
      height: 100%;
      width: 100%;
      padding: 0;
      margin: 0;
      background-color: #f0f0f0;
      font-size: 16px;
      font-family: 'Avenir', Helvetica, Arial, sans-serif;
      -webkit-font-smoothing: antialiased;
      -moz-osx-font-smoothing: grayscale;
      text-align: left;
      color: #333;
      -moz-user-select:none;
      -webkit-user-select:none;
      -ms-user-select:none;
      -khtml-user-select:none;
      user-select:none;
    }
    .cydex-main div {
      padding: 0 !important;
      margin: 0 !important;
    }
    .cydex-main p {
      padding: 0 !important;
      margin: 0 !important;
      text-indent: 2em;
    }
    .cydex-main {
      padding: 20px 30px !important;
      word-wrap: break-word;
      word-break:break-all;
    }
    .cydex-main .letter,.cydex-main .remarks,.details {
    }
    .cydex-main .remarks,.cydex-main .details {
      padding: 12px 0;
      color: #747474
    }
    .cydex-main .remarks {
      /* border-top: 1px solid #ccc; */
      /* margin-top: 20px !important; */
      padding-top: 30px !important;
    }
    .cydex-main .details {
      padding-top: 30px !important;
    }
    .cydex-main .title {
      text-indent: 0;
      text-align: left;
      display: inline-block;
      width: 189px;
      overflow: hidden;
      margin: 0;
    }
  </style>
</head>
<body>
  <div class="cydex-main">
    <div class="letter">
        <div style="font-weight:600;margin-bottom:0px">{{ block "username" .}} {{end}}</div><br>
        <p>{{ block "content" .}}{{end}}</p>
    </div>
    <div class="remarks">
        <div style="font-weight:600;margin-bottom:0px">{{ block "remark_label" .}}{{end}}</div><br>
      <p>{{ block "remark" .}}{{end}}</p>
    </div>
    <div class="details">
      <div style="font-weight:600;margin-bottom:0px">{{ block "detail_label" .}}{{end}}</div><br>
      {{ block "detail" .}} {{end}}
      <!-- <p><span class="title">包裹大小:</span> ddd</p> -->
      <!-- <p><span class="title">Number of files involved:</span> ddd</p> -->
      <!-- <p><span class="title">上传:</span> ddd</p> -->
    </div>
  </div>
</body>
</html>
