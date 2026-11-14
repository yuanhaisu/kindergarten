












<!DOCTYPE html>
<html lang="zh">

<head>

<link rel="stylesheet" type="text/css" href="diet/dist/DataTables/css/dataTables.bootstrap.min.css" />
<link rel="stylesheet" type="text/css" href="diet/dist/Buttons/css/buttons.bootstrap.min.css" />
<link rel="stylesheet" type="text/css" href="diet/dist/FixedHeader/css/fixedHeader.bootstrap.min.css" />
<link rel="stylesheet" type="text/css" href="diet/dist/Responsive/css/responsive.bootstrap.min.css" />
<link href="diet/dist/bootstrap.css" rel="stylesheet">
<link href="diet/dist/paging.css" rel="stylesheet">
<link href="diet/dist/bootstrap-multiselect-0.9.13/css/bootstrap-multiselect.css" rel="stylesheet">
<link href="diet/dist/bootstrap-datepicker-1.6.0/css/bootstrap-datepicker.min.css" rel="stylesheet">
<link href="diet/dist/szbj.css" rel="stylesheet">
<link href="diet/dist/toastr.min.css" rel="stylesheet">
<style>
.text-center {
    text-align: center !important;
    vertical-align: middle !important;
}

.row {
    margin-right: 15px;
    margin-left: -15px;
}
</style>
</head>

<body>
<!--计划食谱营养分析表-->
<div id="wrapper">
<div class="loading" style="position:fixed"></div>
	<div id="page-wrapper">
	  <div class="row">
		<div class="col-lg-12">
		  <div class="panel panel-default">
			<div class="panel-heading">
			  <span style="float: left;line-height: 30px;">日期：</span>
				<div class="ftui-datapick" style="width: 196px;">
					<div class="datapick"></div>
					<input type="hidden" name="birthdayStart" class="slestart" />
					<input type="hidden" name="birthdayEnd" class="sleend" />
					<input id="fiterbirthday" autocomplete="off" readonly="readonly" class="datainp" />
					<div id="data_box" class="data_box" style="display: none;">
						<!--日期区域-->
						<div class="data_main">
							<!--查询按钮区域-->
							<span class="main_top">
								<p>日期范围</p>
								<div class="data_but">
							   		<input class="but_sty but_true" type="button" callfun="initObjDefine()" value="提交" onclick="datasubmit(this)"/>
							   		<input class="but_sty but_of" type="button" value="取消" onclick="datacancel()"/>
								</div>
							</span>
							<!--日期控件区域-->
							<div class="control">
								<div class="picker lf">
									<input type="text" id="startdata" class="showinp startdata">
									<span id="start"></span>
								</div>
								<div class="picker rg">
									<input type="text" id="enddata" class="showinp enddata" readonly="readonly" />
									<span id="end"></span>
								</div>
							</div>
							<div class="clear"></div>
						</div>
						<!--范围选择区域-->
						<div class="data_range">
							<ul class="range_ul">
								<p class="range_title">预设范围</p>
								<li class="check_data" tag='today'>今日</li>
								<li tag='thisweek'>本周</li>
								<li tag='nowmouth'>当前月</li>
							</ul>
						</div>
						<div class="clear"></div>
					</div>
				</div>&emsp;
			  <span>查询方式：<select id="queryType" size='2'></select></span>&emsp;
			  <span id ="s1" style="display: none;">计划食谱名称：<select id="dietDefine" size='2'></select></span>
			  <span id ="s2" style="display: none;">集体对象名称：<select id="objDefine" size='2'></select></span>&emsp;
			  <a href="javascript:void(0)" id="query" class="btn btn-info btn-long">查询</a>&emsp;
			  <a id="download" onclick="downLoadExcel()" class="btn text-green pull-right"><i class="fa fa-file"></i>&nbsp;导出Excel</a>
				<a id="print" onclick="print()" class="btn text-green pull-right"><i class="fa fa-print"></i>&nbsp;打印</a>
			</div>
			<h4 id="noRecipe" class="text-center" style="display:none">没有食谱信息！
			  <p class="text-center" style="height:300px"></p>
			</h4>
			<div id="hasRecipe" class="panel-body" style="display:none">
			  <ul class="nav-diet nav-tabs-diet" id="myTab">
				<li value="analysisTable" class="active"><a href="javascript:void(0)" data-toggle="tab">计划食谱营养分析表</a></li>
				<li value="scgnb"><a href="javascript:void(0)" data-toggle="tab">计划食谱三餐供能比</a></li>
				<li value="ssgcb"><a href="javascript:void(0)" data-toggle="tab">计划食谱膳食构成表</a></li>
				<li value="caFeFrom"><a href="javascript:void(0)" data-toggle="tab">计划食谱钙铁来源</a></li>
			  </ul>
			  <div class="tab-content">
				<div class="tab-pane fade in active" id="analysisTable">
				  <div>
				      <h3 class='text-center'>计划食谱营养分析表
				      </h3>
				      <span class="pull-right"><a href="javascript:void(0)" id="showChart" class="btn btn-info btn-long"><i class="fa fa-pie-chart"></i>&nbsp;分析图</a></span>
				  </div>
				  <br>
				  <br>
				  <span>&emsp;填报单位：<span id="kinderName"></span></span>
				  <span class="pull-right">报表日期：<span id="reportDate"></span>&emsp;</span>
				  <br>
				  <br>
				  <div >
					<table id="nutritionTable" class="table table-bordered table-hover text-center" style="width:100%">
					  <thead>
						<tr><th colspan=15 class="text-center">营养摄入量</th></tr>
						<tr>
						  <th></th>
						  <th>热量<br>(千卡)</th>
						  <th>热量<br>(千焦)</th>
						  <th>蛋白质<br>(克)</th>
						  <th>脂肪<br>(克)</th>
						  <th>碳水化合物<br>(克)</th>
						  <th>视黄醇当量<br>(微克)</th>
						  <th>维生素A<br>(微克)</th>
						  <th>胡萝卜素<br>(微克)</th>
						  <th>维生素B1<br>(毫克)</th>
						  <th>维生素B2<br>(毫克)</th>
						  <th>维生素C<br>(毫克)</th>
						  <th>钙<br>(毫克)</th>
						  <th>锌<br>(毫克)</th>
						  <th>铁<br>(毫克)</th>
						</tr>
					  </thead>
					  <tbody>
					  </tbody>
					</table>
				  </div>
				  <div >
					<table id="energyTable" class="table table-bordered">
					  <thead>
					    <tr>
						  <th rowspan=2 colspan=2 style="vertical-align: middle;">热量来源</th>
						  <th colspan=2>脂肪</th>
						  <th colspan=2>蛋白质</th>
						  <th colspan=2>碳水化合物</th></tr>
						<tr>
						  <th>要求</th>
						  <th>现状</th>
						  <th>要求</th>
						  <th>现状</th>
						  <th>要求</th>
						  <th>现状</th>
						</tr>
					  </thead>
					  <tbody>
					  </tbody>
					</table>
				  </div>
				  <div style="float:left;width:48%">
					<table id="proteinTable" class="table table-bordered">
					  <thead>
					    <tr>
						  <th colspan=4 class="text-center">蛋白质来源</th></tr>
						<tr>
						  <th rowspan=2></th>
						  <th colspan=3>优质蛋白质</th></tr>
						<tr>
						  <th>要求</th>
						  <th>动物性食物</th>
						  <th>豆类</th></tr>
						<tr>
					  </thead>
					  <tbody>
					  </tbody>
					</table>
				  </div>
				  <div style="float:right;width:48%">
					<table id="fatsTable" class="table table-bordered">
					  <thead>
					    <tr>
						  <th colspan=3 class="text-center">脂肪来源</th>
						</tr>
						<tr>
						  <th></th>
						  <th>动物性</th>
						  <th>植物性</th>
						</tr>
					  </thead>
					  <tbody>
					  </tbody>
					</table>
				  </div>
				  <div class="clearfix"></div>
				  <div>
					  <div class="dataTable_wrapper" style="width:100%">
						<h3 class="text-center">定性分析结论</h3>
						<table id="tableAnalysis" class="table table-striped table-hover">
						  <thead></thead>
						  <tbody></tbody>
						</table>
					  </div>
				  </div>
				</div>
				<div class="tab-content">
					<div class="tab-pane fade in" id="analysisChart">
					  <div>
					      <h3 class='text-center'>计划食谱营养分析图
					      </h3>
					      <span class="pull-right"><a href="javascript:void(0)" id="showTable" class="btn btn-info btn-long"><i class="fa fa-bar-chart"></i>&nbsp;分析表</a></span>
					  </div>
					  <br>
					  <br>
					  <div >
				         <span>&emsp;填报单位：<span id="kinderName"></span></span>
				         <span class="pull-right">报表日期：<span id="reportDate"></span>&emsp;</span>
				      </div>
					  <div  id="bar">
					     <div ><!-- class="panel-body text-center pie-border" -->
						     <h4 class="text-center">营养摄入量</h4>
						     <div id="nutritionCanvas" style="width:100%;height:500px;"></div>
						   </div>
					  </div>
					  <div  id="pie">
						  <div style="width:30%;float:left;">
						      <div class="panel panel-default">
						          <div class="panel-heading">
						              <h4 class="text-center">热量营养素来源分布（千卡）</h4>
						          </div>
								  <div class="panel-body text-center pie-border">
								     <div id="energyCanvas" style="width:100%;height:400px;"></div>
								 </div>
							  </div>
						  </div>
						  <div style="width:30%;float:left;margin-left:5%">
						      <div class="panel panel-default">
						          <div class="panel-heading">
						              <h4 class="text-center">蛋白质来源分布（克）</h4>
						          </div>
								  <div class="panel-body text-center pie-border">
								     <div id="proteinCanvas" style="width:100%;height:400px;"></div>
								 </div>
							  </div>
						  </div>
						  <div style="width:30%;float:right;">
						      <div class="panel panel-default">
						          <div class="panel-heading">
						              <h4 class="text-center">动物性脂肪（克）</h4>
						          </div>
								  <div class="panel-body text-center pie-border">
								     <div id="fatsCanvas" style="width:100%;height:400px;"></div>
								 </div>
							  </div>
						  </div>
						</div>
					</div>
				</div>
				<div class="tab-content">
				  <div class="tab-pane fade in" id="scgnb">
					<div><h3 class='text-center'>计划食谱三餐供能比</h3></div>
					<span>&emsp;填报单位：<span id="kinderName"></span></span>
					<span class="pull-right">报表日期：<span id="reportDate"></span>&emsp;</span>
					<br>
				    <br>
					<div >
					  <table id="scgnbWeekTable" class="table table-bordered table-hover text-center">
						<thead>
						  <tr>
							<th>类别</th>
							<td>早餐</td>
							<td>早点</td>
							<td>午餐</td>
							<td>午点</td>
							<td>晚餐</td>
							<td>晚点</td>
						  </tr>
						</thead>
						<tbody></tbody>
					  </table>
					</div>
				  </div>
				</div>
				<div class="tab-content">
				  <div class="tab-pane fade in" id="ssgcb">
					<div><h3 class='text-center'>计划食谱膳食构成表</h3></div>
					<span>&emsp;填报单位：<span id="kinderName"></span></span>
					<span class="pull-right">报表日期：<span id="reportDate"></span>&emsp;</span>
					<br>
					<br>
					<div >
					  <table id="tableSsgcbRpt" class="table table-bordered table-hover text-center">
						<thead>
						  <tr>
							<th>类别</th>
							<th>谷类</th>
							<th>薯类</th>
							<th>蔬菜</th>
							<th>水果</th>
							<th>畜禽肉类</th>
							<th>蛋类</th>
							<th>水产类</th>
							<th>大豆</th>
							<th>乳制品</th>
							<th>油脂类</th>
						 </tr>
						</thead>
						<tbody></tbody>
					  </table>
					</div>
				  </div>
				</div>
				<div class="tab-content">
				  <div class="tab-pane fade in" id="caFeFrom">
					<div><h3 class='text-center'>计划食谱钙铁来源</h3></div>
					<span>&emsp;填报单位：<span id="kinderName"></span></span>
					<span class="pull-right">报表日期：<span id="reportDate"></span>&emsp;</span>
					<br>
					<br>
					<div >
					  <table id="tableTglyRpt" class="table table-bordered table-hover text-center">
					    <thead>
					      <tr>
					        <th colspan=4>钙</th>
					        <th colspan=2>铁</th>
					      </tr>
					      <tr>
					        <th>来源</th>
					        <td>植物性</td>
					        <td>动物性</td>
					        <td>奶类及制品</td>
					        <td>肉类</td>
					        <td>非肉类</td>
					      </tr>
					    </thead>
					    <tbody></tbody>
					  </table>
					</div>
				  </div>
				</div>
			  </div>
			</div>
		  </div>
		</div>
	  </div>
	</div>
</div>














<!--[if lt IE 9]>
	<script src="/bower_components/html5shiv/dist/html5shiv.min.js?v=v1.01888"></script>
    <script src="/bower_components/respond/dest/respond.min.js?v=v1.01888"></script>
<![endif]-->


<script >
$(document).unbind('click');
</script>

<script src="/diet/dist/bootstrap.min.js?v=v1.01888"></script>


<script src="/diet/dist/bootstrap-multiselect-0.9.13/js/bootstrap-multiselect.min.js?v=v1.01888"></script>


<script src="/diet/dist/bootstrap-datepicker-1.6.0/js/bootstrap-datepicker.min.js?v=v1.01888"></script>
<script src="/diet/dist/bootstrap-datepicker-1.6.0/locales/bootstrap-datepicker.zh-CN.min.js?v=v1.01888"></script>


<script src="/diet/dist/szbj.js?v=v1.01888"></script>
<script src="/diet/dist/tipbox.js?v=v1.01888"></script>
<script src="/diet/dist/toastr.min.js?v=v1.01888"></script>













<script type="text/javascript" src="/diet/dist/JSZip/jszip.min.js?v=v1.01888"></script>
<script type="text/javascript" src="/diet/dist/DataTables/js/jquery.dataTables.min.js?v=v1.01888"></script>
<script type="text/javascript" src="/diet/dist/DataTables/js/dataTables.bootstrap.min.js?v=v1.01888"></script>
<script type="text/javascript" src="/diet/dist/Buttons/js/dataTables.buttons.min.js?v=v1.01888"></script>
<script type="text/javascript" src="/diet/dist/Buttons/js/buttons.bootstrap.min.js?v=v1.01888"></script>
<script type="text/javascript" src="/diet/dist/Buttons/js/buttons.colVis.min.js?v=v1.01888"></script>
<script type="text/javascript" src="/diet/dist/Buttons/js/buttons.html5.min.js?v=v1.01888"></script>
<script type="text/javascript" src="/diet/dist/Buttons/js/buttons.print.min.js?v=v1.01888"></script>
<script type="text/javascript" src="/diet/dist/FixedHeader/js/dataTables.fixedHeader.min.js?v=v1.01888"></script>
<script type="text/javascript" src="/diet/dist/Responsive/js/dataTables.responsive.min.js?v=v1.01888"></script>
<script type="text/javascript" src="/diet/dist/echarts-all-3.js?v=v1.01888"></script>
   
<script type="text/javascript">
var nutritionBarData;
var energyPieData;
var proteinPieData;
var fatsPieData;
$(document).ready(function() {
	if(!window.sessionStorage){
	    alert('您的浏览器版本过低，请升级您的浏览器版本！',"info"); /* IE8以上支持 */
	    return;
	}
	
	//初始化日期
	initDate();
	
	initSelect();

	$("#myTab>li").on('click',function() {
		$(".tab-pane").removeClass("active");
		$("#"+$(this).attr("value")).addClass("active");
	});
	$("#showChart").on('click',function() {
		$(".tab-pane").removeClass("active");
		$("#analysisChart").addClass("active");
		initAllchart();
	});
	$("#showTable").on('click',function() {
		$(".tab-pane").removeClass("active");
		$("#analysisTable").addClass("active");
	});
	$("#query").on('click',function() {
		queryAnalysis();
	});
});

$(window).resize(function() {
	//$("#proteinTable").css("height",$("#percentTable").outerHeight(true)-20);
	$("#fatsTable").css("height",$("#proteinTable").outerHeight(true)-20);
	outerWidth();
	// 如果当前 tab 为膳食营养分析图  窗口调整 则重画图
	if($("#analysisChart").hasClass("active")) {
		initAllchart();
	}
});

function outerWidth() {
	var windowWidth  = $(window).outerWidth(true);
	if(windowWidth < 1236) {
		if(windowWidth < 980) {$('body').css("font-size",'12px');}
		$('.table > thead > tr > th').css("padding",'8px 0px 8px 0px');
		$('.table > tbody > tr > th').css("padding",'8px 0px 8px 0px');
		$('.table > tfoot > tr > th').css("padding",'8px 0px 8px 0px');
		$('.table > thead > tr > td').css("padding",'8px 0px 8px 0px');
		$('.table > tbody > tr > td').css("padding",'8px 0px 8px 0px');
		$('.table > tfoot > tr > td').css("padding",'8px 0px 8px 0px');
	} else {
		if(windowWidth > 980) {$('body').css("font-size",'14px');}
		$('.table > thead > tr > th').css("padding",'8px');
		$('.table > tbody > tr > th').css("padding",'8px');
		$('.table > tfoot > tr > th').css("padding",'8px');
		$('.table > thead > tr > td').css("padding",'8px');
		$('.table > tbody > tr > td').css("padding",'8px');
		$('.table > tfoot > tr > td').css("padding",'8px');
	}
}
var queryTypeList = [{
		label : "按集体对象名称查询",
		value : 2
	},{
		label : "按计划食谱名称查询",
		value : 1
	}];
var objList = [];
var dietList = [];
var sourceID = null;

function initSelect(){	
	$('#queryType').multiselect({
		nonSelectedText: '请选择查询方式',
		disableIfEmpty : true,
		onChange : function(option, checked){
			initObjDefine();
		}
	});
	$('#queryType').multiselect('dataprovider', queryTypeList);
	var loadType = sessionStorage.getItem("queryType");
	sessionStorage.removeItem("queryType");
	if(loadType == 1){
		$('#queryType').multiselect('select', 1, true);
	}else{
		$('#queryType').multiselect('select', 2, true);
	}
	
	$('#dietDefine').multiselect({
		nonSelectedText: '请选择计划食谱',
		disableIfEmpty : true,
		onChange: function(option, checked) {
			console.log(option.text())
			$.each(dietList,function(i,s){
				if(s.value == option.val()){
					sourceID = s.sourceID;
					$('#fiterbirthday').val(s.startDate+'~'+s.endDate);
					$("#fiterbirthday").parent().children('.slestart').val(s.startDate);
					$("#fiterbirthday").parent().children('.sleend').val(s.endDate);
				}
			});
		}
	});
	
	$('#objDefine').multiselect({
		nonSelectedText: '请选择集体对象',
		disableIfEmpty : true,
	});
}

function genParam(){
	var queryType = $('#queryType').val();
	var objID = null;
	if(queryType == 2){
		objID = $('#objDefine').val();
	}else if(queryType == 1 ){
		objID = sourceID;
	}else{
		setWorn('请选择查询方式');
		return null;
	}
	if(objID == null || objID < 0){
		if(queryType == 1){
			setWorn('请选择计划食谱');
		}else{
			setWorn('请选择集体对象');
		}
		return null;
	}
	if($('.slestart').val()==""){
		setWorn('请选择日期');
		return null;
	}
	var param = {
		'startDate':$('.slestart').val(),
		'endDate':$('.sleend').val(),
		'queryType':queryType,
		'objID':objID,
	}
	return param;
}

function initDate(){
	$.ajax({
		type: "post",
		url: "/diet/getPlanDietDates.do",
		async: false,
		data: setPojo({}),
		contentType: 'application/json',
		success: function(json) {
			if(json.success == true) {
				dataCaledar(null,json.body);
			}
		}
	});

	var loadType = sessionStorage.getItem("queryType");
	if(loadType == 1){
		var startD = sessionStorage.getItem("startDate");
		sessionStorage.removeItem("startDate");
		var endD = sessionStorage.getItem("endDate");
		sessionStorage.removeItem("endDate");
		$('#fiterbirthday').val(startD+'~'+endD);
		$("#fiterbirthday").parent().children('.slestart').val(startD);
		$("#fiterbirthday").parent().children('.sleend').val(endD);
	}else{
		$('#fiterbirthday').val(getLastDayNew(0,true)+'~'+getLastDayNew(0));
		$("#fiterbirthday").parent().children('.slestart').val(getLastDayNew(0,true));
		$("#fiterbirthday").parent().children('.sleend').val(getLastDayNew(0));
	}
}
var isFirst = true;

//加载集体对象
function initObjDefine(){
	var queryType = $('#queryType').val();
	if(queryType == 2){
		
	}else if(queryType == 1){

	}else{
		setWorn('请选择查询方式');
		return false;
	}
	$.ajax({
		async:true,
		url:'/diet/report/GeneralDietTypeBeforeReport/initQuery.do',
		contentType:'application/json',
		data:JSON.stringify({
			startDate:$('.slestart').val(),
			endDate:$('.sleend').val(),
			queryType:$('#queryType').val()
		}),
		success:function(data){
			if(data.success==true){
				if(queryType == 1){
					dietList = [];
					sourceID = null;
					$.each(data.body.list,function(index,model) {						
						dietList.push({
							label : model.dietName,
							value : model.dietID,
							sourceID : model.sourceID,
							startDate : model.startDate,
							endDate: model.endDate,
						});
					});
					$('#dietDefine').multiselect('dataprovider', dietList);
					
					var dietID = sessionStorage.getItem("dietID");
					sessionStorage.removeItem("dietID");
					if(dietID > 0){
						$("#dietDefine").multiselect('select',dietID,true);
						$("#dietDefine").multiselect('refresh');
						$("#noRecipe").hide();
						if(isFirst) {
							queryAnalysis();
							isFirst = false;
						} 
					}else{
						if(data.body.list.length==0){// 0个食谱 隐藏
							$('#dietDefine').multiselect('refresh');
							$("#noRecipe").show();
							$("#hasRecipe").hide();
						}else if(data.body.list.length==1){ // 一个食谱直接选中
							$("#dietDefine").multiselect('select',dietList[0].value,true);
							$("#dietDefine").multiselect('refresh');
							$("#noRecipe").hide();
							if(isFirst) {
								queryAnalysis();
								isFirst = false;
							} 
						}else{ // 多个集体对象 用户自己选
							$("#dietDefine").multiselect('deselectAll',false);
							$("#dietDefine").multiselect('updateButtonText');
							$("#noRecipe").hide();
						}
					}
					$('#s2').hide();
					$('#s1').show();	
				}else{
					objList = [];
					$.each(data.body.list,function(index,model) {						
						objList.push({
							label : model.objName,
							value : model.objID
						});
					});
					$('#objDefine').multiselect('dataprovider', objList);
					if(data.body.list.length==0){// 0个集体对象 隐藏
						$('#objDefine').multiselect('refresh');
						$("#noRecipe").show();
						$("#hasRecipe").hide();
					}else if(data.body.list.length==1){ // 一个集体对象直接选中
						$("#objDefine").multiselect('select',objList[0].value,true);
						$("#objDefine").multiselect('refresh');
						$("#noRecipe").hide();
						if(isFirst) {
							queryAnalysis();
							isFirst = false;
						} 
					}else{ // 多个集体对象 用户自己选
						$("#objDefine").multiselect('deselectAll',false);
						$("#objDefine").multiselect('updateButtonText');
						$("#noRecipe").hide();
					}
					$('#s1').hide();
					$('#s2').show();
				}
			}else{
				displayNew(data.msg);
			}
		},
		error:function(data){
			displayNew(null);
		}
	});
}

function queryAnalysis() {
	var param = genParam();
	if(param == null){
		return false;	
	}
	$.ajax({
		async:true,
		url:'/diet/report/GeneralDietTypeBeforeReport/query.do',
		contentType:'application/json',
		beforeSend: function(){
			 loadingOpen("dataloading");
		},
		data:JSON.stringify(param),
		success:function(data){
			if(data.success==true){
				if(data.body == null) {
					noRecipe.style.display = 'block';
					hasRecipe.style.display = 'none';
					loadingClose();
					return false;
				}
				noRecipe.style.display = 'none';
				hasRecipe.style.display = 'block';
				var text = formatDateRange($('.slestart').val(),$('.sleend').val());
				$(document).find('span[id="reportDate"]').text(text);
				// 显示 营养摄入量
				showNutrition(data.body.nutrition);
				// 显示能量来源
				showEnergy(data.body.energy);
				// 显示 蛋白质
				showProtein(data.body.dbz);
				// 显示 脂肪来源
				showFats(data.body.fats);
				// 显示 定性分析结论
				showAnalysis(data.body.analysis);
				//显示 三餐供能比 (今日)
				//showScgnbDay(data.body.scgnbDay);
				// 显示 三餐供能比 (本周)
				showScgnbWeek(data.body.scgnb);
				// 显示 膳食构成比
				showSsgcb(data.body.ssgcb);
				// 显示 钙铁来源
				showFeAndCaFrom(data.body.tgly);
				$("#fatsTable").css("height",$("#proteinTable").outerHeight(true)-20);
				initAllchart();
				outerWidth();
				$(document).find('span[id="kinderName"]').text(data.body.orgName);
				loadingClose();
			} else {
				loadingClose();
				displayNew(data.msg);
			}
		},
		error:function(data){
			loadingClose();
			displayNew(null);
		}
	});
}

// 显示 营养摄入量
function showNutrition(list) {
	var trs = '';
	$.each(list,function(inde,model) {
		for(var key in model) {
			if(!isNaN(model[key])) {
				model[key] = toFixed(model[key],2);
			}
		}
		trs += '<tr><th>'+model.label+'</th><td>'+model.energy+'</td><td>'+model.heat+'</td><td>'+model.protein+'</td><td>'+
		model.fats+'</td><td>'+model.carbohydrate+'</td><td>'+model.vitaminA+'</td><td>'+model.vitaminA+'</td><td>'+
		'-'+'</td><td>'+model.vitaminB1+'</td><td>'+model.vitaminB2+'</td><td>'+model.vitaminC+'</td><td>'+
		model.ca+'</td><td>'+model.zn+'</td><td>'+model.fe+'</td></tr>';
	});
	// 显示表
	$("#nutritionTable>tbody").html(trs);
	// 显示图
	nutritionBarData = list[2];
}

//显示　能量来源
function showEnergy(list) {
	var trs = '';
	$.each(list,function(index,model) {
		for(var key in model) {
			if(!isNaN(model[key])) {
				model[key] = toFixed(model[key],2);
			}
		}
		trs += '<tr>index='+index+'<td>'+model.fatsRecommend+'</td><td>'+model.fatsIntake+'</td><td>'+model.proteinRecommend+
		'</td><td>'+model.proteinIntake+'</td><td>'+model.carbohydrateRecommend+'</td><td>'+model.carbohydrateIntake+'</td></tr>';
	});
	trs = trs.replace(/index=0/,'<th rowspan=2>摄入量</th><th>千卡</th>');
	trs = trs.replace(/index=1/,'<th>千焦</th>');
	trs = trs.replace(/index=2/,'<th colspan=2>占总热量%</th>');
	trs = trs.replace(/index=3/,'<th colspan=2>营养素比例</th>');
	$('#energyTable>tbody').html(trs);
	var model = list[0];
	energyPieData = [];
	energyPieData.push(model.fatsIntake);
	energyPieData.push(model.proteinIntake);
	energyPieData.push(model.carbohydrateIntake);
}

// 显示　蛋白质来源
function showProtein(list) {
	var trs = '';
	$.each(list,function(index,model) {
		trs += '<tr><th>'+model.label+'</th><td>'+model.require+'</td><td>'+toFixed(model.animalprotein,2)+'</td><td>'+
		toFixed(model.beanprotein,2)+'</td></tr>';
	});
	$('#proteinTable>tbody').html(trs);
	var model = list[0];
	proteinPieData = [];
	proteinPieData.push(toFixed(model.animalprotein,2));
	proteinPieData.push(toFixed(model.beanprotein,2));
	proteinPieData.push(toFixed(parseFloat(model.require)*2-parseFloat(proteinPieData[0])-parseFloat(proteinPieData[1]),2));
}

//　显示　脂肪来源
function showFats(list) {
	var trs = '';
	$.each(list,function(index,model) {
		trs += '<tr><th>'+model.label+'</th><td>'+toFixed(model.animalfats,2)+'</td><td>'+toFixed(model.vegfats,2)+'</td></tr>';
	});
	$('#fatsTable>tbody').html(trs);
	var model = list[0];
	fatsPieData = [];
	fatsPieData.push(toFixed(model.animalfats,2));
	fatsPieData.push(toFixed(model.vegfats,2));
	fatsPieData.push(toFixed(model.otherfats,2));
}

// 显示 定性分析结论
function showAnalysis(analysis) {
	$("#tableAnalysis>tbody").html('<tr><th>'+analysis+'</th></tr>')
}

//显示 三餐供能比 (今日)
function showScgnbDay(list) {
	var checkDateToday = $("#checkDate").val().split("-");
	$("#reportDateToday").text(checkDateToday[0]+"年"+checkDateToday[1]+"月"+checkDateToday[2]+"日");
	if(list == undefined || list == null || list.length == 0){
		$("#scgnbDateTable>tbody").html('<tr><th colspan=7 class="text-center">今日无食谱信息！</th></tr>');
		return false;
	}
	var trs = '';
	$.each(list,function(index,model) {
		for(var key in model) {
			if(!isNaN(model[key])) {
				model[key] = toFixed(model[key],2);
			}
		}
		trs += '<tr><th>'+model.label+'</th><td>'+model[1]+'</td><td>'+model[2]+'</td><td>'+model[4]+
		'</td><td>'+model[8]+'</td><td>'+model[16]+'</td><td>'+model[32]+'</td></tr>';
	});
	$("#scgnbDateTable>tbody").html(trs);
}

// 显示 三餐供能比 (本周)
function showScgnbWeek(list) {
	var trs = '';
	$.each(list,function(index,model) {
		for(var key in model) {
			if(!isNaN(model[key])) {
				model[key] = toFixed(model[key],2);
			}
		}
		trs += '<tr><th>'+model.label+'</th><td>'+model[1]+'</td><td>'+model[2]+'</td><td>'+model[4]+
		'</td><td>'+model[8]+'</td><td>'+model[16]+'</td><td>'+model[32]+'</td></tr>';
	});
	$("#scgnbWeekTable>tbody").html(trs);
}

// 显示 膳食构成表
function showSsgcb(model) {
	//var allHaveValue = false;
	for(var item in model) {
		model[item] = toFixed(model[item],2);
		/* if(model[item] != 0) {
			allHaveValue = true;
		} */
	}
	if(true/* allHaveValue */) {
		var tr = '<tr><th>摄入量(克)</th><td>'+model['谷类']+'</td><td>'+model['薯类']+'</td><td>'+model['蔬菜']+'</td><td>'+model['水果']+'</td><td>'+model['畜禽肉类']+'</td><td>'
		+model['蛋类']+'</td><td>'+model['水产类']+'</td><td>'+model['大豆']+'</td><td>'+model['乳制品']+'</td><td>'+model['油脂类']+'</td></tr>';
		$('#tableSsgcbRpt>tbody').html(tr);
	} else {
		$('#tableSsgcbRpt>tbody').empty();
	}
}

// 显示 铁钙来源
function showFeAndCaFrom(list) {
	if(list.length > 0) {
		$.each(list,function(index,model) {
			for(var item in model) {
				if(item != 'label') {
					model[item] = toFixed(model[item],2);
				}
			}
		});
		var intakeModel = list[0];
		// 钙
		var intakeTrs = '<tr><th>'+intakeModel.label+'</th><td>'+intakeModel.vegca+'</td><td>'+
		intakeModel.animalca+'</td><td>'+intakeModel.milk+'</td>';
		// 铁
		intakeTrs += '<td class="border-left">'+intakeModel.meatfe+'</td><td>'+intakeModel.notmeatfe+'</td></tr>';
		var constituteModel = list[1];
		// 钙
		var constituteTrs = '<tr><th>'+constituteModel.label+'</th><td>'+constituteModel.vegca+'</td><td>'+
		constituteModel.animalca+'</td><td>'+constituteModel.milk+'</td>';
		// 铁
		constituteTrs += '<td class="border-left">'+constituteModel.meatfe+'</td><td>'+constituteModel.notmeatfe+'</td></tr>';
		$('#tableTglyRpt>tbody').html(intakeTrs + constituteTrs);
	} else {
		$('#tableTglyRpt>tbody').empty()
	}
}

// 加载所有图图
function initAllchart(){
	initNutritionBarData(nutritionBarData);
	initEnergyPieData(energyPieData);
	initProteinPieData(proteinPieData);
	initFatsPieData(fatsPieData);
}

// 能量来源饼图
function initEnergyPieData(list){
	/* var legendFats = '脂肪'+list[0]+'千卡';
	var legendProtein = '蛋白质'+list[1]+'千卡';
	var legendCarbohydrate = '碳水化合物'+list[2]+'千卡';*/
	var legendData = ['脂肪','蛋白质','碳水化合物']; 
	var data = [
				{value:list[0], name:'脂肪', itemStyle:{normal:{color:'#9bbb58'}}},// 绿色
                {value:list[1], name:'蛋白质', itemStyle:{normal:{color:'#c0504e'}}},// 红色
                {value:list[2], name:'碳水化合物', itemStyle:{normal:{color:'#4f81bc'}}}// 蓝色
            ];
	showPie('energy', legendData, data);
}

// 蛋白质来源饼图
function initProteinPieData(list){
	var legendData = ['动物性食物','豆类','其他'];
	var data = [
                {value:list[0], name:'动物性食物', itemStyle:{normal:{color:'#c0504e'}}},// 红色
                {value:list[1], name:'豆类', itemStyle:{normal:{color:'#9bbb58'}}},// 绿色
                {value:list[2], name:'其他', itemStyle:{normal:{color:'#4f81bc'}}}// 蓝色
            ];
	showPie('protein', legendData, data);
}

// 脂肪来源饼图
function initFatsPieData(list){
	var legendData = ['动物性','植物性','其他'];
	var data = [
                {value:list[0], name:'动物性', itemStyle:{normal:{color:'#9bbb58'}}},// 绿色
                {value:list[1], name:'植物性', itemStyle:{normal:{color:'#c0504e'}}},// 红色
                {value:list[2], name:'其他', itemStyle:{normal:{color:'#4f81bc'}}}// 红色
            ];
	showPie('fats', legendData, data);
}

// 饼图显示方法
function showPie(id, legendData, data){
    // 指定图表的配置项和数据
    var option = {
        title : { // 是否显示 标题
        	show: false
	    },
	    tooltip : { // 鼠标 指 到某个上面显示 内容 
	        trigger: 'item',
	        formatter: "{b}:{d}%({c}克)" // 格式化显示数据  {a}（系列名称），{b}（类目值），{c}（数值）
	    },
	    legend: {// 图例
	        orient: 'vertical',
	        bottom: 'bottom',
	        data: legendData
	    },
	    series : [
	        {
	            type: 'pie',
	            radius : '50%',//半径长度px
	            center: ['50%', '50%'], //中心位置[x,y]
	            data: data,
	            //roseType : 'angle',
	            itemStyle: {
	                emphasis: {
	                    shadowBlur: 10,
	                    shadowOffsetX: 0,
	                    shadowColor: 'rgba(0, 0, 0, 0.5)'
	                },
	                normal:{ 
	                    label:{ 
	                      show: true, 
	                      formatter: '{b}:\n{d}%\n({c}克)'
	                    }, 
	                    labelLine :{show:true} 
	                }
	            }
	        }
	    ]
    };
    // 使用刚指定的配置项和数据显示图表。
    window.setTimeout(function(){
    	echarts.init(document.getElementById(id+'Canvas')).setOption(option);
    }, 100);
}

// 脂肪来源饼图
function initNutritionBarData(model){
	var legendData = '';
	var data = [];
	data.push(model.energy);
	data.push(model.heat);
	data.push(model.protein);
	data.push(model.fats);
	data.push(model.carbohydrate);
	data.push(model.vitaminA);
	data.push(model.vitaminA);
	//data.push(model.carotin);
	data.push(model.vitaminB1);
	data.push(model.vitaminB2);
	data.push(model.vitaminC);
	data.push(model.ca);
	data.push(model.zn);
	data.push(model.fe);
	showBar('nutrition', legendData, data);
}

// 柱状图显示方法
function showBar(id,legendData,data) {
	/* var windowWidth  = $(window).outerWidth(true);
	var rotate;
	var y2;
	if(windowWidth > 1236) {
		
	} */
	//rotate
	var option = {
		    title: { //  显示 标题
		       /*  textAlign:'center',
		        text: '营养素摄入量' */
		    	how: false
		    },
		    color: ['#5bc0de'],// 全局柱状图颜色
		    tooltip: {// 鼠标 指 到某个上面显示 内容 
		         formatter: '{a}:{c}%'// 格式化显示数据  {a}（系列名称），{b}（类目值），{c}（数值）
		    },
		    xAxis: {
		        data: ["热量(千卡)","热量(千焦)","蛋白质(克)","脂肪(克)","碳水化合物(克)"
		        ,"视黄醇当量(微克)","维生素A(微克)","维生素B1(毫克)","维生素B2(毫克)",
		        "维生素C(毫克)","钙(微克)","锌(微克)","铁(微克)"],
		        axisLabel:{interval: 0, rotate: 45}// 防止 x坐标显示丢失，但是屏幕过小会重叠
		    },
		    grid:{
		    	y2:100
		    },
		    yAxis: {
		        axisLabel : {
		                formatter: '{value} .00%'// 纵坐标格式化显示数据 
		            }
		    },
		    series: [{
		        name: '百分比',
		        type: 'bar',
		        data: data
		    }]
		};
	// 使用刚指定的配置项和数据显示图表。
    window.setTimeout(function(){
    	var chart = echarts.init(document.getElementById(id+'Canvas'));
    	chart.setOption(option);
    	//$(window).resize(function(){ chart.resize(); });
    }, 100);
	
}

//日期格式化
Date.prototype.Format = function(fmt) {
	var o = { 
			 "M+" : this.getMonth()+1,//月份 
			 "d+" : this.getDate()//日 
			};
	if(/(y+)/.test(fmt)) {
		fmt=fmt.replace(RegExp.$1, (this.getFullYear()+"").substr(4 - RegExp.$1.length)); 
	} 
	for(var k in o) {
		if(new RegExp("("+ k +")").test(fmt)) {
			fmt = fmt.replace(RegExp.$1, (RegExp.$1.length==1) ? (o[k]) : (("00"+ o[k]).substr((""+ o[k]).length))); 
		}
	}
	return fmt;
}

function downLoadExcel() {
	var param = genParam();
	if(param==null){
		return false;
	}
	dietExportXls(Purls["khs"]+'/diet/report/GeneralDietTypeBeforeReport/export.do', 
			setPojo(param), "计划食谱营养分析");
}

function print() {
	var param = genParam();
	if(param==null){
		return false;
	}
	$.tipBox.confirm("请在浏览器设置中允许弹出窗口<br>推荐打印设置为A4横向打印", {
		buttons: [{
			'打印': function() {
				$.tipBox.confirm.close();
				$.ajax({
					async:true,
					url:'/diet/report/GeneralDietTypeBeforeReport/query.do',
					contentType:'application/json',
					beforeSend: function(){
					},
					data:JSON.stringify(param),
					success:function(data){
						if(data.success==true){
							sessionStorage.setItem('printData',JSON.stringify(data));
							window.open("/diet/report/overall/recipe/printRecipeWithAmoutAnalysis.jsp");
						}
					}
				});
			}
		}]
	});
}

//输出日期范围
function formatDateRange(startDate,endDate){
	var start = new Date(startDate);
	var end = new Date(endDate);
	if(start.getFullYear()==end.getFullYear()){
		return start.Format("yyyy年MM月dd日")+"-"+end.Format("MM月dd日");
	}else{
		return start.Format("yyyy年MM月dd日")+"-"+end.Format("yyyy年MM月dd日");
	}
}
</script>
</body>

</html>