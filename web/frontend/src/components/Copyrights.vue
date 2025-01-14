<template>
    <v-footer padless>
        <v-row justify="center" no-gutters>
            <v-col style="font-size: 14px;" class="py-4 text-center white--text" cols="12">
                © {{ new Date().getFullYear() }} | <strong>Powered by
                <a class="white--text" href="https://cardinal.ink/" target="_blank">Cardinal</a> | Modified with 😡 by Pim
            </strong>
            </v-col>
        </v-row>
    </v-footer>

</template>

<script>
    export default {
        name: "Copyrights"
    }
</script>

<style scoped>

</style>