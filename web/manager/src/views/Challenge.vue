<template>    
    <div>
        <div class="challengeVisible" style="text-align:center;color:gray;font-size:18px;">
            <p>A challenge can only be set to visible if there is a matching GameBox.</p>
        </div>
        <el-button type="primary" @click="newChallengeDialogVisible = true">{{$t('challenge.new')}}</el-button>
        <el-table :data="challengeList" style="width: 100%" stripe v-loading="isLoading">
            <el-table-column width="80" prop="ID" label="ID"/>
            <el-table-column prop="Title" :label="$t('challenge.title')"/>
            <el-table-column prop="BaseScore" :label="$t('challenge.base_score')"/>
            <el-table-column prop="Visible" :label="$t('challenge.visible')">
                <template slot-scope="scope">{{scope.row.Visible}}</template>
            </el-table-column>
            <el-table-column prop="AutoRefreshFlag" :label="$t('challenge.auto_refresh_flag')">
                <template slot-scope="scope">{{scope.row.AutoRefreshFlag}}</template>
            </el-table-column>
            <el-table-column prop="Command" :label="$t('challenge.command')"/>
            <el-table-column prop="CheckdownCommand" :label="$t('challenge.checkdown_command')"/>
            <el-table-column :label="$t('general.create_at')" width="200"
                             :formatter="(row)=>utils.FormatGoTime(row.CreatedAt)"/>
            <el-table-column :label="$t('general.operate')" width="300">
                <template slot-scope="scope">
                    <el-button plain size="mini" slot="reference" @click="handleVisible(scope.row.ID, !scope.row.Visible)">{{scope.row.Visible ? $t('challenge.invisible_title') : $t('challenge.visible_title')}}</el-button>
                    <el-button
                            size="mini"
                            @click="()=>{editChallengeForm = JSON.parse(JSON.stringify(scope.row)); editChallengeDialogVisible = true}">
                        {{$t('general.edit')}}
                    </el-button>
                    <el-button size="mini" type="danger" slot="reference" @click="handleDelete(scope.row)">{{$t('general.delete')}}
                    </el-button>
                </template>
            </el-table-column>
        </el-table>

        <!-- New Challenge -->
        <el-dialog :title="$t('challenge.publish')" :visible.sync="newChallengeDialogVisible">
            <el-form :model="newChallengeForm" label-width="120px">
                <el-form-item :label="$t('challenge.title')">
                    <el-input v-model="newChallengeForm.Title"/>
                </el-form-item>
                <el-form-item :label="$t('challenge.base_score')">
                    <el-input-number v-model="newChallengeForm.BaseScore" :min="1"/>
                </el-form-item>
                <el-form-item :label="$t('challenge.auto_refresh_flag')">
                    <el-switch v-model="newChallengeForm.AutoRefreshFlag"></el-switch>
                </el-form-item>
                <el-form-item :label="$t('challenge.command')" v-if="newChallengeForm.AutoRefreshFlag">
                    <el-input v-model="newChallengeForm.Command"/>
                    <span>{{$t('challenge.flag_placeholder')}}<code v-pre> {{FLAG}}</code></span>
                </el-form-item>
                <el-form-item :label="$t('challenge.checkdown_command')">
                    <el-input v-model="newChallengeForm.CheckdownCommand"/>
                    <span>{{$t('challenge.check_down_placeholder')}}<code v-pre> {{IP}} & {{PORT}}</code></span>
                </el-form-item>
            </el-form>
            <el-button type="primary" @click="onNewChallenge">{{$t('challenge.publish')}}</el-button>
        </el-dialog>

        <!-- Edit Challenge -->
        <el-dialog :title="$t('challenge.edit')" :visible.sync="editChallengeDialogVisible">
            <el-form :model="editChallengeForm" label-width="120px">
                <el-form-item :label="$t('challenge.title')">
                    <el-input v-model="editChallengeForm.Title"/>
                </el-form-item>
                <el-form-item :label="$t('challenge.base_score')">
                    <el-input-number v-model="editChallengeForm.BaseScore" :min="1"/>
                </el-form-item>
                <el-form-item :label="$t('challenge.auto_refresh_flag')">
                    <el-switch v-model="editChallengeForm.AutoRefreshFlag"></el-switch>
                </el-form-item>
                <el-form-item :label="$t('challenge.command')" v-if="editChallengeForm.AutoRefreshFlag">
                    <el-input v-model="editChallengeForm.Command"/>
                    <span>{{$t('challenge.flag_placeholder')}}<code v-pre>{{FLAG}}</code></span>
                </el-form-item>
                <el-form-item :label="$t('challenge.checkdown_command')">
                    <el-input v-model="editChallengeForm.CheckdownCommand"/>
                    <span>{{$t('challenge.check_down_placeholder')}}<code v-pre> {{IP}} & {{PORT}}</code></span>
                </el-form-item>
            </el-form>
            <el-button type="primary" @click="onEditChallenge">{{$t('challenge.edit')}}</el-button>
        </el-dialog>

    </div>
</template>

<script>
    export default {
        name: "Challenge",
        data() {
            return {
                isLoading: true,
                challengeList: null,
                newChallengeDialogVisible: false,
                editChallengeDialogVisible: false,

                newChallengeForm: {
                    Title: '',
                    BaseScore: 1000,
                    AutoRefreshFlag: false,
                    Command: 'echo "{{FLAG}}" > /flag',
                    CheckdownCommand: '',
                },

                editChallengeForm: {
                    Title: '',
                    BaseScore: 1000,
                    AutoRefreshFlag: false,
                    Command: '',
                    CheckdownCommand: '',
                },
            }
        },

        mounted() {
            this.getChallenges()
        },

        methods: {
            getChallenges() {
                this.utils.GET("/manager/challenges").then(res => {
                    this.challengeList = res
                    this.isLoading = false
                }).catch(err => this.$message.error(err))
            },

            onNewChallenge() {
                this.utils.POST('/manager/challenge', this.newChallengeForm).then(res => {
                    this.newChallengeDialogVisible = false
                    // Clear the form
                    this.newChallengeForm = {
                        Title: '',
                        BaseScore: 1000,
                        AutoRefreshFlag: false,
                        Command: 'echo "{{FLAG}}" > /flag',
                        CheckdownCommand: '',
                    }
                    this.getChallenges()
                    this.$message({message: res, type: 'success'})
                }).catch(err => this.$message({message: err, type: 'error'}))
            },

            onEditChallenge() {
                this.utils.PUT('/manager/challenge', this.editChallengeForm).then(res => {
                    this.editChallengeDialogVisible = false
                    this.getChallenges()
                    this.$message({message: res, type: 'success'})
                }).catch(err => this.$message({message: err, type: 'error'}))
            },

            handleDelete(row) {
                this.utils.DELETE("/manager/challenge?id=" + row.ID).then(res => {
                    this.$message({
                        message: res,
                        type: 'success'
                    });
                    this.getChallenges()
                }).catch(err => this.$message({message: err, type: 'error'}))
            },

            handleVisible(id, visible) {
                this.utils.POST("/manager/challenge/visible", {
                    ID: id,
                    Visible: visible
                }).then(res => {
                    this.$message({
                        message: res,
                        type: 'success'
                    });
                    this.getChallenges()
                }).catch(err => this.$message({message: err, type: 'error'}))
            }
        }
    }
</script>

<style scoped>

</style>
